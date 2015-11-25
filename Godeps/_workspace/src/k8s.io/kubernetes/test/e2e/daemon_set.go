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
	"reflect"
	"strings"
	"time"

	"k8s.io/kubernetes/pkg/api"
	apierrs "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/experimental"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/kubectl"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/util/wait"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	// this should not be a multiple of 5, because node status updates
	// every 5 seconds. See https://github.com/kubernetes/kubernetes/pull/14915.
	updateRetryPeriod    = 2 * time.Second
	updateRetryTimeout   = 30 * time.Second
	daemonsetLabelPrefix = "daemonset-"
	daemonsetNameLabel   = daemonsetLabelPrefix + "name"
	daemonsetColorLabel  = daemonsetLabelPrefix + "color"
)

var _ = Describe("Daemon set", func() {
	f := &Framework{BaseName: "daemonsets"}

	BeforeEach(func() {
		f.beforeEach()
		err := clearDaemonSetNodeLabels(f.Client)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := clearDaemonSetNodeLabels(f.Client)
		Expect(err).NotTo(HaveOccurred())
		f.afterEach()
	})

	It("should launch a daemon pod on every node of the cluster", func() {
		testDaemonSets(f)
	})
})

func separateDaemonSetNodeLabels(labels map[string]string) (map[string]string, map[string]string) {
	daemonSetLabels := map[string]string{}
	otherLabels := map[string]string{}
	for k, v := range labels {
		if strings.HasPrefix(k, daemonsetLabelPrefix) {
			daemonSetLabels[k] = v
		} else {
			otherLabels[k] = v
		}
	}
	return daemonSetLabels, otherLabels
}

func clearDaemonSetNodeLabels(c *client.Client) error {
	nodeClient := c.Nodes()
	nodeList, err := nodeClient.List(labels.Everything(), fields.Everything())
	if err != nil {
		return err
	}
	for _, node := range nodeList.Items {
		_, err := setDaemonSetNodeLabels(c, node.Name, map[string]string{})
		if err != nil {
			return err
		}
	}
	return nil
}

func setDaemonSetNodeLabels(c *client.Client, nodeName string, labels map[string]string) (*api.Node, error) {
	nodeClient := c.Nodes()
	var newNode *api.Node
	var newLabels map[string]string
	err := wait.Poll(updateRetryPeriod, updateRetryTimeout, func() (bool, error) {
		node, err := nodeClient.Get(nodeName)
		if err != nil {
			return false, err
		}

		// remove all labels this test is creating
		daemonSetLabels, otherLabels := separateDaemonSetNodeLabels(node.Labels)
		if reflect.DeepEqual(daemonSetLabels, labels) {
			newNode = node
			return true, nil
		}
		node.Labels = otherLabels
		for k, v := range labels {
			node.Labels[k] = v
		}
		newNode, err = nodeClient.Update(node)
		if err == nil {
			newLabels, _ = separateDaemonSetNodeLabels(newNode.Labels)
			return true, err
		}
		if se, ok := err.(*apierrs.StatusError); ok && se.ErrStatus.Reason == unversioned.StatusReasonConflict {
			Logf("failed to update node due to resource version conflict")
			return false, nil
		}
		return false, err
	})
	if err != nil {
		return nil, err
	} else if len(newLabels) != len(labels) {
		return nil, fmt.Errorf("Could not set daemon set test labels as expected.")
	}

	return newNode, nil
}

func checkDaemonPodOnNodes(f *Framework, selector map[string]string, nodeNames []string) func() (bool, error) {
	return func() (bool, error) {
		podList, err := f.Client.Pods(f.Namespace.Name).List(labels.Set(selector).AsSelector(), fields.Everything())
		if err != nil {
			return false, nil
		}
		pods := podList.Items

		nodesToPodCount := make(map[string]int)
		for _, pod := range pods {
			nodesToPodCount[pod.Spec.NodeName] += 1
		}

		// Ensure that exactly 1 pod is running on all nodes in nodeNames.
		for _, nodeName := range nodeNames {
			if nodesToPodCount[nodeName] != 1 {
				return false, nil
			}
		}

		// Ensure that sizes of the lists are the same. We've verified that every element of nodeNames is in
		// nodesToPodCount, so verifying the lengths are equal ensures that there aren't pods running on any
		// other nodes.
		return len(nodesToPodCount) == len(nodeNames), nil
	}
}

func checkRunningOnAllNodes(f *Framework, selector map[string]string) func() (bool, error) {
	return func() (bool, error) {
		nodeList, err := f.Client.Nodes().List(labels.Everything(), fields.Everything())
		if err != nil {
			return false, nil
		}
		nodeNames := make([]string, 0)
		for _, node := range nodeList.Items {
			nodeNames = append(nodeNames, node.Name)
		}
		return checkDaemonPodOnNodes(f, selector, nodeNames)()
	}
}

func checkRunningOnNoNodes(f *Framework, selector map[string]string) func() (bool, error) {
	return checkDaemonPodOnNodes(f, selector, make([]string, 0))
}

func testDaemonSets(f *Framework) {
	ns := f.Namespace.Name
	c := f.Client
	simpleDSName := "simple-daemon-set"
	image := "gcr.io/google_containers/serve_hostname:1.1"
	label := map[string]string{daemonsetNameLabel: simpleDSName}
	retryTimeout := 1 * time.Minute
	retryInterval := 5 * time.Second

	Logf("Creating simple daemon set %s", simpleDSName)
	_, err := c.DaemonSets(ns).Create(&experimental.DaemonSet{
		ObjectMeta: api.ObjectMeta{
			Name: simpleDSName,
		},
		Spec: experimental.DaemonSetSpec{
			Template: &api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: label,
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  simpleDSName,
							Image: image,
							Ports: []api.ContainerPort{{ContainerPort: 9376}},
						},
					},
				},
			},
		},
	})
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		Logf("Check that reaper kills all daemon pods for %s", simpleDSName)
		dsReaper, err := kubectl.ReaperFor("DaemonSet", c)
		Expect(err).NotTo(HaveOccurred())
		_, err = dsReaper.Stop(ns, simpleDSName, 0, nil)
		Expect(err).NotTo(HaveOccurred())
		err = wait.Poll(retryInterval, retryTimeout, checkRunningOnNoNodes(f, label))
		Expect(err).NotTo(HaveOccurred(), "error waiting for daemon pod to be reaped")
	}()

	By("Check that daemon pods launch on every node of the cluster.")
	Expect(err).NotTo(HaveOccurred())
	err = wait.Poll(retryInterval, retryTimeout, checkRunningOnAllNodes(f, label))
	Expect(err).NotTo(HaveOccurred(), "error waiting for daemon pod to start")

	By("Stop a daemon pod, check that the daemon pod is revived.")
	podClient := c.Pods(ns)

	podList, err := podClient.List(labels.Set(label).AsSelector(), fields.Everything())
	Expect(err).NotTo(HaveOccurred())
	Expect(len(podList.Items)).To(BeNumerically(">", 0))
	pod := podList.Items[0]
	err = podClient.Delete(pod.Name, nil)
	Expect(err).NotTo(HaveOccurred())
	err = wait.Poll(retryInterval, retryTimeout, checkRunningOnAllNodes(f, label))
	Expect(err).NotTo(HaveOccurred(), "error waiting for daemon pod to revive")

	complexDSName := "complex-daemon-set"
	complexLabel := map[string]string{daemonsetNameLabel: complexDSName}
	nodeSelector := map[string]string{daemonsetColorLabel: "blue"}
	Logf("Creating daemon with a node selector %s", complexDSName)
	_, err = c.DaemonSets(ns).Create(&experimental.DaemonSet{
		ObjectMeta: api.ObjectMeta{
			Name: complexDSName,
		},
		Spec: experimental.DaemonSetSpec{
			Selector: complexLabel,
			Template: &api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: complexLabel,
				},
				Spec: api.PodSpec{
					NodeSelector: nodeSelector,
					Containers: []api.Container{
						{
							Name:  complexDSName,
							Image: image,
							Ports: []api.ContainerPort{{ContainerPort: 9376}},
						},
					},
				},
			},
		},
	})
	Expect(err).NotTo(HaveOccurred())

	By("Initially, daemon pods should not be running on any nodes.")
	err = wait.Poll(retryInterval, retryTimeout, checkRunningOnNoNodes(f, complexLabel))
	Expect(err).NotTo(HaveOccurred(), "error waiting for daemon pods to be running on no nodes")

	By("Change label of node, check that daemon pod is launched.")
	nodeClient := c.Nodes()
	nodeList, err := nodeClient.List(labels.Everything(), fields.Everything())
	Expect(len(nodeList.Items)).To(BeNumerically(">", 0))
	newNode, err := setDaemonSetNodeLabels(c, nodeList.Items[0].Name, nodeSelector)
	Expect(err).NotTo(HaveOccurred(), "error setting labels on node")
	daemonSetLabels, _ := separateDaemonSetNodeLabels(newNode.Labels)
	Expect(len(daemonSetLabels)).To(Equal(1))
	err = wait.Poll(retryInterval, retryTimeout, checkDaemonPodOnNodes(f, complexLabel, []string{newNode.Name}))
	Expect(err).NotTo(HaveOccurred(), "error waiting for daemon pods to be running on new nodes")

	By("remove the node selector and wait for daemons to be unscheduled")
	_, err = setDaemonSetNodeLabels(c, nodeList.Items[0].Name, map[string]string{})
	Expect(err).NotTo(HaveOccurred(), "error removing labels on node")
	Expect(wait.Poll(retryInterval, retryTimeout, checkRunningOnNoNodes(f, complexLabel))).
		NotTo(HaveOccurred(), "error waiting for daemon pod to not be running on nodes")

	By("We should now be able to delete the daemon set.")
	Expect(c.DaemonSets(ns).Delete(complexDSName)).NotTo(HaveOccurred())
}
