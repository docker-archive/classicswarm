// +build linux

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

package kubelet

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/docker/libcontainer/cgroups"
	"github.com/docker/libcontainer/cgroups/fs"
	"github.com/docker/libcontainer/configs"
	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/kubelet/cadvisor"
	"k8s.io/kubernetes/pkg/util"
	"k8s.io/kubernetes/pkg/util/errors"
	"k8s.io/kubernetes/pkg/util/mount"
	"k8s.io/kubernetes/pkg/util/oom"
	"k8s.io/kubernetes/pkg/util/sets"
)

const (
	// The percent of the machine memory capacity. The value is used to calculate
	// docker memory resource container's hardlimit to workaround docker memory
	// leakage issue. Please see kubernetes/issues/9881 for more detail.
	DockerMemoryLimitThresholdPercent = 70
	// The minimum memory limit allocated to docker container: 150Mi
	MinDockerMemoryLimit = 150 * 1024 * 1024
)

// A non-user container tracked by the Kubelet.
type systemContainer struct {
	// Absolute name of the container.
	name string

	// CPU limit in millicores.
	cpuMillicores int64

	// Function that ensures the state of the container.
	// m is the cgroup manager for the specified container.
	ensureStateFunc func(m *fs.Manager) error

	// Manager for the cgroups of the external container.
	manager *fs.Manager
}

func newSystemContainer(containerName string) *systemContainer {
	return &systemContainer{
		name:    containerName,
		manager: createManager(containerName),
	}
}

type nodeConfig struct {
	dockerDaemonContainerName string
	systemContainerName       string
	kubeletContainerName      string
}

type containerManagerImpl struct {
	cadvisorInterface cadvisor.Interface
	mountUtil         mount.Interface
	nodeConfig
	// External containers being managed.
	systemContainers []*systemContainer
}

var _ containerManager = &containerManagerImpl{}

// checks if the required cgroups subsystems are mounted.
// As of now, only 'cpu' and 'memory' are required.
func validateSystemRequirements(mountUtil mount.Interface) error {
	const (
		cgroupMountType = "cgroup"
		localErr        = "system validation failed"
	)
	mountPoints, err := mountUtil.List()
	if err != nil {
		return fmt.Errorf("%s - %v", localErr, err)
	}
	expectedCgroups := sets.NewString("cpu", "cpuacct", "cpuset", "memory")
	for _, mountPoint := range mountPoints {
		if mountPoint.Type == cgroupMountType {
			for _, opt := range mountPoint.Opts {
				if expectedCgroups.Has(opt) {
					expectedCgroups.Delete(opt)
				}
			}
		}
	}

	if expectedCgroups.Len() > 0 {
		return fmt.Errorf("%s - Following Cgroup subsystem not mounted: %v", localErr, expectedCgroups.List())
	}
	return nil
}

// TODO(vmarmol): Add limits to the system containers.
// Takes the absolute name of the specified containers.
// Empty container name disables use of the specified container.
func newContainerManager(mountUtil mount.Interface, cadvisorInterface cadvisor.Interface, dockerDaemonContainerName, systemContainerName, kubeletContainerName string) (containerManager, error) {
	return &containerManagerImpl{
		cadvisorInterface: cadvisorInterface,
		mountUtil:         mountUtil,
		nodeConfig: nodeConfig{
			dockerDaemonContainerName: dockerDaemonContainerName,
			systemContainerName:       systemContainerName,
			kubeletContainerName:      kubeletContainerName,
		},
	}, nil
}

// Create a cgroup container manager.
func createManager(containerName string) *fs.Manager {
	return &fs.Manager{
		Cgroups: &configs.Cgroup{
			Name:            containerName,
			AllowAllDevices: true,
		},
	}
}

func (cm *containerManagerImpl) setupNode() error {
	if err := validateSystemRequirements(cm.mountUtil); err != nil {
		return err
	}
	systemContainers := []*systemContainer{}
	if cm.dockerDaemonContainerName != "" {
		cont := newSystemContainer(cm.dockerDaemonContainerName)

		info, err := cm.cadvisorInterface.MachineInfo()
		var capacity = api.ResourceList{}
		if err != nil {
		} else {
			capacity = CapacityFromMachineInfo(info)
		}
		memoryLimit := (int64(capacity.Memory().Value() * DockerMemoryLimitThresholdPercent / 100))
		if memoryLimit < MinDockerMemoryLimit {
			glog.Warningf("Memory limit %d for container %s is too small, reset it to %d", memoryLimit, cm.dockerDaemonContainerName, MinDockerMemoryLimit)
			memoryLimit = MinDockerMemoryLimit
		}

		glog.V(2).Infof("Configure resource-only container %s with memory limit: %d", cm.dockerDaemonContainerName, memoryLimit)

		dockerContainer := &fs.Manager{
			Cgroups: &configs.Cgroup{
				Name:            cm.dockerDaemonContainerName,
				Memory:          memoryLimit,
				MemorySwap:      -1,
				AllowAllDevices: true,
			},
		}
		cont.ensureStateFunc = func(manager *fs.Manager) error {
			return ensureDockerInContainer(cm.cadvisorInterface, -900, dockerContainer)
		}
		systemContainers = append(systemContainers, cont)
	}

	if cm.systemContainerName != "" {
		if cm.systemContainerName == "/" {
			return fmt.Errorf("system container cannot be root (\"/\")")
		}

		rootContainer := &fs.Manager{
			Cgroups: &configs.Cgroup{
				Name: "/",
			},
		}
		manager := createManager(cm.systemContainerName)

		err := ensureSystemContainer(rootContainer, manager)
		if err != nil {
			return err
		}
		systemContainers = append(systemContainers, newSystemContainer(cm.systemContainerName))
	}

	if cm.kubeletContainerName != "" {
		systemContainers = append(systemContainers, newSystemContainer(cm.kubeletContainerName))
	}
	cm.systemContainers = systemContainers
	return nil
}

func (cm *containerManagerImpl) Start() error {
	// Setup the node
	if err := cm.setupNode(); err != nil {
		return err
	}
	// Don't run a background thread if there are no ensureStateFuncs.
	numEnsureStateFuncs := 0
	for _, cont := range cm.systemContainers {
		if cont.ensureStateFunc != nil {
			numEnsureStateFuncs++
		}
	}
	if numEnsureStateFuncs == 0 {
		return nil
	}

	// Run ensure state functions every minute.
	go util.Until(func() {
		for _, cont := range cm.systemContainers {
			if cont.ensureStateFunc != nil {
				if err := cont.ensureStateFunc(cont.manager); err != nil {
					glog.Warningf("[ContainerManager] Failed to ensure state of %q: %v", cont.name, err)
				}
			}
		}
	}, time.Minute, util.NeverStop)

	return nil
}

func (cm *containerManagerImpl) SystemContainersLimit() api.ResourceList {
	cpuLimit := int64(0)

	// Sum up resources of all external containers.
	for _, cont := range cm.systemContainers {
		cpuLimit += cont.cpuMillicores
	}

	return api.ResourceList{
		api.ResourceCPU: *resource.NewMilliQuantity(
			cpuLimit,
			resource.DecimalSI),
	}
}

// Ensures that the Docker daemon is in the desired container.
func ensureDockerInContainer(cadvisor cadvisor.Interface, oomScoreAdj int, manager *fs.Manager) error {
	// What container is Docker in?
	out, err := exec.Command("pidof", "docker").Output()
	if err != nil {
		return fmt.Errorf("failed to find pid of Docker container: %v", err)
	}

	// The output of pidof is a list of pids.
	// Docker may be forking and thus there would be more than one result.
	pids := []int{}
	for _, pidStr := range strings.Split(strings.TrimSpace(string(out)), " ") {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}

	// Move if the pid is not already in the desired container.
	errs := []error{}
	for _, pid := range pids {
		cont, err := getContainer(pid)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to find container of PID %d: %v", pid, err))
		}

		if cont != manager.Cgroups.Name {
			err = manager.Apply(pid)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to move PID %d (in %q) to %q", pid, cont, manager.Cgroups.Name))
			}
		}

		// Also apply oom-score-adj to processes
		oomAdjuster := oom.NewOOMAdjuster()
		if err := oomAdjuster.ApplyOOMScoreAdj(pid, oomScoreAdj); err != nil {
			errs = append(errs, fmt.Errorf("failed to apply oom score %d to PID %d", oomScoreAdj, pid))
		}
	}

	return errors.NewAggregate(errs)
}

// Gets the (CPU) container the specified pid is in.
func getContainer(pid int) (string, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return "", err
	}
	defer f.Close()

	return cgroups.ParseCgroupFile("cpu", f)
}

// Ensures the system container is created and all non-kernel threads and process 1
// without a container are moved to it.
//
// The reason of leaving kernel threads at root cgroup is that we don't want to tie the
// execution of these threads with to-be defined /system quota and create priority inversions.
//
// The reason of leaving process 1 at root cgroup is that libcontainer hardcoded on
// the base cgroup path based on process 1. Please see:
// https://github.com/kubernetes/kubernetes/issues/12789#issuecomment-132384126
// for detail explanation.
func ensureSystemContainer(rootContainer *fs.Manager, manager *fs.Manager) error {
	// Move non-kernel PIDs to the system container.
	attemptsRemaining := 10
	var errs []error
	for attemptsRemaining >= 0 {
		// Only keep errors on latest attempt.
		errs = []error{}
		attemptsRemaining--

		allPids, err := rootContainer.GetPids()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to list PIDs for root: %v", err))
			continue
		}

		// Remove kernel pids and process 1
		pids := make([]int, 0, len(allPids))
		for _, pid := range allPids {
			if isKernelPid(pid) {
				continue
			}

			// TODO(dawnchen): Remove this once the hard dependency on process 1 is removed
			// on systemd node.
			if pid == 1 {
				continue
			}
			pids = append(pids, pid)
		}
		glog.Infof("Found %d PIDs in root, %d of them are kernel related", len(allPids), len(allPids)-len(pids))

		// Check if we moved all the non-kernel PIDs.
		if len(pids) == 0 {
			break
		}

		glog.Infof("Moving non-kernel threads: %v", pids)
		for _, pid := range pids {
			err := manager.Apply(pid)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to move PID %d into the system container %q: %v", pid, manager.Cgroups.Name, err))
				continue
			}
		}

	}
	if attemptsRemaining < 0 {
		errs = append(errs, fmt.Errorf("ran out of attempts to create system containers %q", manager.Cgroups.Name))
	}

	return errors.NewAggregate(errs)
}

// Determines whether the specified PID is a kernel PID.
func isKernelPid(pid int) bool {
	// Kernel threads have no associated executable.
	_, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	return err != nil
}
