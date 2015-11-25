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

package container

import (
	"hash/adler32"
	"strings"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util"
	"k8s.io/kubernetes/third_party/golang/expansion"

	"github.com/golang/glog"
)

// HandlerRunner runs a lifecycle handler for a container.
type HandlerRunner interface {
	Run(containerID string, pod *api.Pod, container *api.Container, handler *api.Handler) error
}

// RunContainerOptionsGenerator generates the options that necessary for
// container runtime to run a container.
type RunContainerOptionsGenerator interface {
	GenerateRunContainerOptions(pod *api.Pod, container *api.Container) (*RunContainerOptions, error)
}

// Trims runtime prefix from ID or image name (e.g.: docker://busybox -> busybox).
func TrimRuntimePrefix(fullString string) string {
	const prefixSeparator = "://"

	idx := strings.Index(fullString, prefixSeparator)
	if idx < 0 {
		return fullString
	}
	return fullString[idx+len(prefixSeparator):]
}

// ShouldContainerBeRestarted checks whether a container needs to be restarted.
// TODO(yifan): Think about how to refactor this.
func ShouldContainerBeRestarted(container *api.Container, pod *api.Pod, podStatus *api.PodStatus, readinessManager *ReadinessManager) bool {
	podFullName := GetPodFullName(pod)

	// Get all dead container status.
	var resultStatus []*api.ContainerStatus
	for i, containerStatus := range podStatus.ContainerStatuses {
		if containerStatus.Name == container.Name && containerStatus.State.Terminated != nil {
			resultStatus = append(resultStatus, &podStatus.ContainerStatuses[i])
		}
	}

	// Set dead containers to notReady state.
	for _, c := range resultStatus {
		readinessManager.RemoveReadiness(TrimRuntimePrefix(c.ContainerID))
	}

	// Check RestartPolicy for dead container.
	if len(resultStatus) > 0 {
		if pod.Spec.RestartPolicy == api.RestartPolicyNever {
			glog.V(4).Infof("Already ran container %q of pod %q, do nothing", container.Name, podFullName)
			return false
		}
		if pod.Spec.RestartPolicy == api.RestartPolicyOnFailure {
			// Check the exit code of last run. Note: This assumes the result is sorted
			// by the created time in reverse order.
			if resultStatus[0].State.Terminated.ExitCode == 0 {
				glog.V(4).Infof("Already successfully ran container %q of pod %q, do nothing", container.Name, podFullName)
				return false
			}
		}
	}
	return true
}

// HashContainer returns the hash of the container. It is used to compare
// the running container with its desired spec.
func HashContainer(container *api.Container) uint64 {
	hash := adler32.New()
	util.DeepHashObject(hash, *container)
	return uint64(hash.Sum32())
}

// EnvVarsToMap constructs a map of environment name to value from a slice
// of env vars.
func EnvVarsToMap(envs []EnvVar) map[string]string {
	result := map[string]string{}
	for _, env := range envs {
		result[env.Name] = env.Value
	}

	return result
}

func ExpandContainerCommandAndArgs(container *api.Container, envs []EnvVar) (command []string, args []string) {
	mapping := expansion.MappingFuncFor(EnvVarsToMap(envs))

	if len(container.Command) != 0 {
		for _, cmd := range container.Command {
			command = append(command, expansion.Expand(cmd, mapping))
		}
	}

	if len(container.Args) != 0 {
		for _, arg := range container.Args {
			args = append(args, expansion.Expand(arg, mapping))
		}
	}

	return command, args
}
