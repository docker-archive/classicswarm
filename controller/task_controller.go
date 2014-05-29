package controller

import (
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"log"
	"math/rand"
	"strconv"
	"strings"
)

type Port struct {
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	HostPort int `yaml:"hostPort,omitempty" json:"hostPort,omitempty"`
	ContainerPort int `yaml:"containerPort,omitempty" json:"containerPort,omitempty"`
}

type EnvVar struct {
	Name string  `yaml:"name,omitempty" json:"name,omitempty"`
	Value string `yaml:"value,omitempty" json:"value,omitempty"`
}

type VolumeMount struct {
	Name      string `yaml:"name,omitempty" json:"name,omitempty"`
	ReadOnly  bool   `yaml:"readOnly,omitempty" json:"readOnly,omitempty"`
	MountPath string `yaml:"mountPath,omitempty" json:"mountPath,omitempty"`
}

type Container struct {
	Name         string        `yaml:"name,omitempty" json:"name,omitempty"`
	Image        string        `yaml:"image,omitempty" json:"image,omitempty"`
	Command      string        `yaml:"command,omitempty" json:"command,omitempty"`
	WorkingDir   string        `yaml:"workingDir,omitempty" json:"workingDir,omitempty"`
	Ports        []Port        `yaml:"ports,omitempty" json:"ports,omitempty"`
	Env          []EnvVar      `yaml:"env,omitempty" json:"env,omitempty"`
	Memory       int           `yaml:"memory,omitempty" json:"memory,omitempty"`
	CPU          int           `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	VolumeMounts []VolumeMount `yaml:"volumeMounts,omitempty" json:"volumeMounts,omitempty"`
}

type TaskController struct {
	client *docker.Client
	containers []Container
}

func MakeTaskController(client *docker.Client, containers []Container) *TaskController {
	return &TaskController{
		client: client,
		containers: containers,
	}
}

func (t *TaskController) RunContainer(container Container) error {
	var err error

	name := fmt.Sprintf("%s-%x", container.Name, rand.Uint32())
	envVariables := []string{}
	for key, value := range container.Env {
		envVariables = append(envVariables, fmt.Sprintf("%s=%s", key, value))
	}

	volumes := map[string]struct{}{}
	binds := []string{}
	for _, volume := range container.VolumeMounts {
		volumes[volume.MountPath] = struct{}{}
		basePath := "/exports/" + volume.Name + ":" + volume.MountPath
		if volume.ReadOnly {
			basePath += ":ro"
		}
		binds = append(binds, basePath)
	}

	exposedPorts := map[docker.Port]struct{}{}
	portBindings := map[docker.Port][]docker.PortBinding{}
	for _, port := range container.Ports {
		interiorPort := port.ContainerPort
		exteriorPort := port.HostPort
		// Some of this port stuff is under-documented voodoo.
		// See http://stackoverflow.com/questions/20428302/binding-a-port-to-a-host-interface-using-the-rest-api
		dockerPort := docker.Port(strconv.Itoa(interiorPort) + "/tcp")
		exposedPorts[dockerPort] = struct{}{}
		portBindings[dockerPort] = []docker.PortBinding{
			docker.PortBinding{
				HostPort: strconv.Itoa(exteriorPort),
			},
		}
	}
	var cmdList []string
	if len(container.Command) > 0 {
		cmdList = strings.Split(container.Command, " ")
	}
	dockerContainer, err := t.client.CreateContainer(docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			Image:        container.Image,
			ExposedPorts: exposedPorts,
			Env:          envVariables,
			Volumes:      volumes,
			WorkingDir:   container.WorkingDir,
			Cmd:          cmdList,
		},
	})
	if err != nil {
		return err
	}
	return t.client.StartContainer(dockerContainer.ID, &docker.HostConfig{
		PortBindings: portBindings,
		Binds:        binds,
	})
}

func (t *TaskController) IsContainerRunning(container Container) (bool, error) {
	containers, err := t.client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return false, err
	}
	for _, listContainer := range containers {
		if listContainer.ID == container.Name {
			return true, nil
		}
	}
	return false, nil
}

func (t *TaskController) SyncContainers() {
	for _, container := range t.containers {
		running, err := t.IsContainerRunning(container)
		if err != nil {
			log.Printf("Error syncing container: %#v (%#v)", container, err)
			continue
		}
		if !running {
			t.RunContainer(container)
		}
	}
}
