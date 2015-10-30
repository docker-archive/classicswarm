/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"fmt"
	math_rand "math/rand"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/latest"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/api/unversioned"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/cloudprovider/providers/aws"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/util"
)

const (
	gcePDDetachTimeout  = 10 * time.Minute
	gcePDDetachPollTime = 10 * time.Second
)

var _ = Describe("Pod Disks", func() {
	var (
		podClient client.PodInterface
		host0Name string
		host1Name string
	)
	framework := NewFramework("pod-disks")

	BeforeEach(func() {
		SkipUnlessNodeCountIsAtLeast(2)

		podClient = framework.Client.Pods(framework.Namespace.Name)

		nodes, err := framework.Client.Nodes().List(labels.Everything(), fields.Everything())
		expectNoError(err, "Failed to list nodes for e2e cluster.")

		Expect(len(nodes.Items)).To(BeNumerically(">=", 2), "Requires at least 2 nodes")

		host0Name = nodes.Items[0].ObjectMeta.Name
		host1Name = nodes.Items[1].ObjectMeta.Name

		math_rand.Seed(time.Now().UTC().UnixNano())
	})

	It("should schedule a pod w/ a RW PD, remove it, then schedule it on another host", func() {
		SkipUnlessProviderIs("gce", "gke", "aws")

		By("creating PD")
		diskName, err := createPD()
		expectNoError(err, "Error creating PD")

		host0Pod := testPDPod([]string{diskName}, host0Name, false /* readOnly */, 1 /* numContainers */)
		host1Pod := testPDPod([]string{diskName}, host1Name, false /* readOnly */, 1 /* numContainers */)
		containerName := "mycontainer"

		defer func() {
			By("cleaning up PD-RW test environment")
			// Teardown pods, PD. Ignore errors.
			// Teardown should do nothing unless test failed.
			podClient.Delete(host0Pod.Name, api.NewDeleteOptions(0))
			podClient.Delete(host1Pod.Name, api.NewDeleteOptions(0))
			detachPD(host0Name, diskName)
			detachPD(host1Name, diskName)
			deletePDWithRetry(diskName)
		}()

		By("submitting host0Pod to kubernetes")
		_, err = podClient.Create(host0Pod)
		expectNoError(err, fmt.Sprintf("Failed to create host0Pod: %v", err))

		expectNoError(framework.WaitForPodRunning(host0Pod.Name))

		testFile := "/testpd1/tracker"
		testFileContents := fmt.Sprintf("%v", math_rand.Int())

		expectNoError(framework.WriteFileViaContainer(host0Pod.Name, containerName, testFile, testFileContents))
		Logf("Wrote value: %v", testFileContents)

		By("deleting host0Pod")
		expectNoError(podClient.Delete(host0Pod.Name, api.NewDeleteOptions(0)), "Failed to delete host0Pod")

		By("submitting host1Pod to kubernetes")
		_, err = podClient.Create(host1Pod)
		expectNoError(err, "Failed to create host1Pod")

		expectNoError(framework.WaitForPodRunning(host1Pod.Name))

		v, err := framework.ReadFileViaContainer(host1Pod.Name, containerName, testFile)
		expectNoError(err)
		Logf("Read value: %v", v)

		Expect(strings.TrimSpace(v)).To(Equal(strings.TrimSpace(testFileContents)))

		By("deleting host1Pod")
		expectNoError(podClient.Delete(host1Pod.Name, api.NewDeleteOptions(0)), "Failed to delete host1Pod")

		By(fmt.Sprintf("deleting PD %q", diskName))
		deletePDWithRetry(diskName)

		return
	})

	It("should schedule a pod w/ a readonly PD on two hosts, then remove both.", func() {
		SkipUnlessProviderIs("gce", "gke")

		By("creating PD")
		diskName, err := createPD()
		expectNoError(err, "Error creating PD")

		rwPod := testPDPod([]string{diskName}, host0Name, false /* readOnly */, 1 /* numContainers */)
		host0ROPod := testPDPod([]string{diskName}, host0Name, true /* readOnly */, 1 /* numContainers */)
		host1ROPod := testPDPod([]string{diskName}, host1Name, true /* readOnly */, 1 /* numContainers */)

		defer func() {
			By("cleaning up PD-RO test environment")
			// Teardown pods, PD. Ignore errors.
			// Teardown should do nothing unless test failed.
			podClient.Delete(rwPod.Name, api.NewDeleteOptions(0))
			podClient.Delete(host0ROPod.Name, api.NewDeleteOptions(0))
			podClient.Delete(host1ROPod.Name, api.NewDeleteOptions(0))

			detachPD(host0Name, diskName)
			detachPD(host1Name, diskName)
			deletePDWithRetry(diskName)
		}()

		By("submitting rwPod to ensure PD is formatted")
		_, err = podClient.Create(rwPod)
		expectNoError(err, "Failed to create rwPod")
		expectNoError(framework.WaitForPodRunning(rwPod.Name))
		expectNoError(podClient.Delete(rwPod.Name, api.NewDeleteOptions(0)), "Failed to delete host0Pod")
		expectNoError(waitForPDDetach(diskName, host0Name))

		By("submitting host0ROPod to kubernetes")
		_, err = podClient.Create(host0ROPod)
		expectNoError(err, "Failed to create host0ROPod")

		By("submitting host1ROPod to kubernetes")
		_, err = podClient.Create(host1ROPod)
		expectNoError(err, "Failed to create host1ROPod")

		expectNoError(framework.WaitForPodRunning(host0ROPod.Name))

		expectNoError(framework.WaitForPodRunning(host1ROPod.Name))

		By("deleting host0ROPod")
		expectNoError(podClient.Delete(host0ROPod.Name, api.NewDeleteOptions(0)), "Failed to delete host0ROPod")

		By("deleting host1ROPod")
		expectNoError(podClient.Delete(host1ROPod.Name, api.NewDeleteOptions(0)), "Failed to delete host1ROPod")

		By(fmt.Sprintf("deleting PD %q", diskName))
		deletePDWithRetry(diskName)

		expectNoError(err, "Error deleting PD")
	})

	It("should schedule a pod w/ a RW PD shared between multiple containers, write to PD, delete pod, verify contents, and repeat in rapid succession", func() {
		SkipUnlessProviderIs("gce", "gke", "aws")

		By("creating PD")
		diskName, err := createPD()
		expectNoError(err, "Error creating PD")
		numContainers := 4

		host0Pod := testPDPod([]string{diskName}, host0Name, false /* readOnly */, numContainers)

		defer func() {
			By("cleaning up PD-RW test environment")
			// Teardown pods, PD. Ignore errors.
			// Teardown should do nothing unless test failed.
			podClient.Delete(host0Pod.Name, api.NewDeleteOptions(0))
			detachPD(host0Name, diskName)
			deletePDWithRetry(diskName)
		}()

		fileAndContentToVerify := make(map[string]string)
		for i := 0; i < 3; i++ {
			Logf("PD Read/Writer Iteration #%v", i)
			By("submitting host0Pod to kubernetes")
			_, err = podClient.Create(host0Pod)
			expectNoError(err, fmt.Sprintf("Failed to create host0Pod: %v", err))

			expectNoError(framework.WaitForPodRunning(host0Pod.Name))

			// randomly select a container and read/verify pd contents from it
			containerName := fmt.Sprintf("mycontainer%v", math_rand.Intn(numContainers)+1)
			verifyPDContentsViaContainer(framework, host0Pod.Name, containerName, fileAndContentToVerify)

			// Randomly select a container to write a file to PD from
			containerName = fmt.Sprintf("mycontainer%v", math_rand.Intn(numContainers)+1)
			testFile := fmt.Sprintf("/testpd1/tracker%v", i)
			testFileContents := fmt.Sprintf("%v", math_rand.Int())
			fileAndContentToVerify[testFile] = testFileContents
			expectNoError(framework.WriteFileViaContainer(host0Pod.Name, containerName, testFile, testFileContents))
			Logf("Wrote value: \"%v\" to PD %q from pod %q container %q", testFileContents, diskName, host0Pod.Name, containerName)

			// Randomly select a container and read/verify pd contents from it
			containerName = fmt.Sprintf("mycontainer%v", math_rand.Intn(numContainers)+1)
			verifyPDContentsViaContainer(framework, host0Pod.Name, containerName, fileAndContentToVerify)

			By("deleting host0Pod")
			expectNoError(podClient.Delete(host0Pod.Name, api.NewDeleteOptions(0)), "Failed to delete host0Pod")
		}

		By(fmt.Sprintf("deleting PD %q", diskName))
		deletePDWithRetry(diskName)

		return
	})

	It("should schedule a pod w/two RW PDs both mounted to one container, write to PD, verify contents, delete pod, recreate pod, verify contents, and repeat in rapid succession", func() {
		SkipUnlessProviderIs("gce", "gke", "aws")

		By("creating PD1")
		disk1Name, err := createPD()
		expectNoError(err, "Error creating PD1")
		By("creating PD2")
		disk2Name, err := createPD()
		expectNoError(err, "Error creating PD2")

		host0Pod := testPDPod([]string{disk1Name, disk2Name}, host0Name, false /* readOnly */, 1 /* numContainers */)

		defer func() {
			By("cleaning up PD-RW test environment")
			// Teardown pods, PD. Ignore errors.
			// Teardown should do nothing unless test failed.
			podClient.Delete(host0Pod.Name, api.NewDeleteOptions(0))
			detachPD(host0Name, disk1Name)
			detachPD(host0Name, disk2Name)
			deletePDWithRetry(disk1Name)
			deletePDWithRetry(disk2Name)
		}()

		containerName := "mycontainer"
		fileAndContentToVerify := make(map[string]string)
		for i := 0; i < 3; i++ {
			Logf("PD Read/Writer Iteration #%v", i)
			By("submitting host0Pod to kubernetes")
			_, err = podClient.Create(host0Pod)
			expectNoError(err, fmt.Sprintf("Failed to create host0Pod: %v", err))

			expectNoError(framework.WaitForPodRunning(host0Pod.Name))

			// Read/verify pd contents for both disks from container
			verifyPDContentsViaContainer(framework, host0Pod.Name, containerName, fileAndContentToVerify)

			// Write a file to both PDs from container
			testFilePD1 := fmt.Sprintf("/testpd1/tracker%v", i)
			testFilePD2 := fmt.Sprintf("/testpd2/tracker%v", i)
			testFilePD1Contents := fmt.Sprintf("%v", math_rand.Int())
			testFilePD2Contents := fmt.Sprintf("%v", math_rand.Int())
			fileAndContentToVerify[testFilePD1] = testFilePD1Contents
			fileAndContentToVerify[testFilePD2] = testFilePD2Contents
			expectNoError(framework.WriteFileViaContainer(host0Pod.Name, containerName, testFilePD1, testFilePD1Contents))
			Logf("Wrote value: \"%v\" to PD1 (%q) from pod %q container %q", testFilePD1Contents, disk1Name, host0Pod.Name, containerName)
			expectNoError(framework.WriteFileViaContainer(host0Pod.Name, containerName, testFilePD2, testFilePD2Contents))
			Logf("Wrote value: \"%v\" to PD2 (%q) from pod %q container %q", testFilePD2Contents, disk2Name, host0Pod.Name, containerName)

			// Read/verify pd contents for both disks from container
			verifyPDContentsViaContainer(framework, host0Pod.Name, containerName, fileAndContentToVerify)

			By("deleting host0Pod")
			expectNoError(podClient.Delete(host0Pod.Name, api.NewDeleteOptions(0)), "Failed to delete host0Pod")
		}

		By(fmt.Sprintf("deleting PD1 %q", disk1Name))
		deletePDWithRetry(disk1Name)
		By(fmt.Sprintf("deleting PD2 %q", disk2Name))
		deletePDWithRetry(disk2Name)

		return
	})
})

func deletePDWithRetry(diskName string) {
	var err error
	for start := time.Now(); time.Since(start) < 180*time.Second; time.Sleep(5 * time.Second) {
		if err = deletePD(diskName); err != nil {
			Logf("Couldn't delete PD %q. Sleeping 5 seconds (%v)", diskName, err)
			continue
		}
		Logf("Deleted PD %v", diskName)
		break
	}
	expectNoError(err, "Error deleting PD")
}

func verifyPDContentsViaContainer(f *Framework, podName, containerName string, fileAndContentToVerify map[string]string) {
	for filePath, expectedContents := range fileAndContentToVerify {
		v, err := f.ReadFileViaContainer(podName, containerName, filePath)
		if err != nil {
			Logf("Error reading file: %v", err)
		}
		expectNoError(err)
		Logf("Read file %q with content: %v", filePath, v)
		Expect(strings.TrimSpace(v)).To(Equal(strings.TrimSpace(expectedContents)))
	}
}

func createPD() (string, error) {
	if testContext.Provider == "gce" || testContext.Provider == "gke" {
		pdName := fmt.Sprintf("%s-%s", testContext.prefix, string(util.NewUUID()))

		zone := testContext.CloudConfig.Zone
		// TODO: make this hit the compute API directly instread of shelling out to gcloud.
		err := exec.Command("gcloud", "compute", "--quiet", "--project="+testContext.CloudConfig.ProjectID, "disks", "create", "--zone="+zone, "--size=10GB", pdName).Run()
		if err != nil {
			return "", err
		}
		return pdName, nil
	} else {
		volumes, ok := testContext.CloudConfig.Provider.(aws_cloud.Volumes)
		if !ok {
			return "", fmt.Errorf("Provider does not support volumes")
		}
		volumeOptions := &aws_cloud.VolumeOptions{}
		volumeOptions.CapacityMB = 10 * 1024
		return volumes.CreateVolume(volumeOptions)
	}
}

func deletePD(pdName string) error {
	if testContext.Provider == "gce" || testContext.Provider == "gke" {
		zone := testContext.CloudConfig.Zone

		// TODO: make this hit the compute API directly.
		cmd := exec.Command("gcloud", "compute", "--quiet", "--project="+testContext.CloudConfig.ProjectID, "disks", "delete", "--zone="+zone, pdName)
		data, err := cmd.CombinedOutput()
		if err != nil {
			dataStr := string(data)
			if strings.Contains(dataStr, "was not found") {
				Logf("PD deletion implicitly succeeded because PD %q does not exist.", pdName)
				return nil
			}

			Logf("Error deleting PD: %s (%v)", dataStr, err)
		}
		return err
	} else {
		volumes, ok := testContext.CloudConfig.Provider.(aws_cloud.Volumes)
		if !ok {
			return fmt.Errorf("Provider does not support volumes")
		}
		return volumes.DeleteVolume(pdName)
	}
}

func detachPD(hostName, pdName string) error {
	if testContext.Provider == "gce" || testContext.Provider == "gke" {
		instanceName := strings.Split(hostName, ".")[0]

		zone := testContext.CloudConfig.Zone

		// TODO: make this hit the compute API directly.
		return exec.Command("gcloud", "compute", "--quiet", "--project="+testContext.CloudConfig.ProjectID, "detach-disk", "--zone="+zone, "--disk="+pdName, instanceName).Run()
	} else {
		volumes, ok := testContext.CloudConfig.Provider.(aws_cloud.Volumes)
		if !ok {
			return fmt.Errorf("Provider does not support volumes")
		}
		return volumes.DetachDisk(hostName, pdName)
	}
}

func testPDPod(diskNames []string, targetHost string, readOnly bool, numContainers int) *api.Pod {
	containers := make([]api.Container, numContainers)
	for i := range containers {
		containers[i].Name = "mycontainer"
		if numContainers > 1 {
			containers[i].Name = fmt.Sprintf("mycontainer%v", i+1)
		}

		containers[i].Image = "gcr.io/google_containers/busybox"

		containers[i].Command = []string{"sleep", "6000"}

		containers[i].VolumeMounts = make([]api.VolumeMount, len(diskNames))
		for k := range diskNames {
			containers[i].VolumeMounts[k].Name = fmt.Sprintf("testpd%v", k+1)
			containers[i].VolumeMounts[k].MountPath = fmt.Sprintf("/testpd%v", k+1)
		}

		containers[i].Resources.Limits = api.ResourceList{}
		containers[i].Resources.Limits[api.ResourceCPU] = *resource.NewQuantity(int64(0), resource.DecimalSI)

	}

	pod := &api.Pod{
		TypeMeta: unversioned.TypeMeta{
			Kind:       "Pod",
			APIVersion: latest.GroupOrDie("").Version,
		},
		ObjectMeta: api.ObjectMeta{
			Name: "pd-test-" + string(util.NewUUID()),
		},
		Spec: api.PodSpec{
			Containers: containers,
			NodeName:   targetHost,
		},
	}

	if testContext.Provider == "gce" || testContext.Provider == "gke" {
		pod.Spec.Volumes = make([]api.Volume, len(diskNames))
		for k, diskName := range diskNames {
			pod.Spec.Volumes[k].Name = fmt.Sprintf("testpd%v", k+1)
			pod.Spec.Volumes[k].VolumeSource = api.VolumeSource{
				GCEPersistentDisk: &api.GCEPersistentDiskVolumeSource{
					PDName:   diskName,
					FSType:   "ext4",
					ReadOnly: readOnly,
				},
			}
		}
	} else if testContext.Provider == "aws" {
		pod.Spec.Volumes = make([]api.Volume, len(diskNames))
		for k, diskName := range diskNames {
			pod.Spec.Volumes[k].Name = fmt.Sprintf("testpd%v", k+1)
			pod.Spec.Volumes[k].VolumeSource = api.VolumeSource{
				AWSElasticBlockStore: &api.AWSElasticBlockStoreVolumeSource{
					VolumeID: diskName,
					FSType:   "ext4",
					ReadOnly: readOnly,
				},
			}
		}
	} else {
		panic("Unknown provider: " + testContext.Provider)
	}

	return pod
}

// Waits for specified PD to to detach from specified hostName
func waitForPDDetach(diskName, hostName string) error {
	if testContext.Provider == "gce" || testContext.Provider == "gke" {
		for start := time.Now(); time.Since(start) < gcePDDetachTimeout; time.Sleep(gcePDDetachPollTime) {
			zone := testContext.CloudConfig.Zone

			cmd := exec.Command("gcloud", "compute", "--project="+testContext.CloudConfig.ProjectID, "instances", "describe", "--zone="+zone, hostName)
			data, err := cmd.CombinedOutput()
			if err != nil {
				Logf("Error waiting for PD %q to detach from node %q. 'gcloud compute instances describe' failed with %s (%v)", diskName, hostName, string(data), err)
				return err
			}

			dataStr := strings.ToLower(string(data))
			diskName = strings.ToLower(diskName)
			if !strings.Contains(string(dataStr), diskName) {
				// Specified disk does not appear to be attached to specified node
				Logf("GCE PD %q appears to have successfully detached from %q.", diskName, hostName)
				return nil
			}

			Logf("Waiting for GCE PD %q to detach from %q.", diskName, hostName)
		}

		return fmt.Errorf("Gave up waiting for GCE PD %q to detach from %q after %v", diskName, hostName, gcePDDetachTimeout)
	}

	return nil
}
