package cluster

import "github.com/samalba/dockerclient"

// Container is exported
type Container struct {
	dockerclient.Container

	Config *ContainerConfig
	Info   dockerclient.ContainerInfo
	Engine *Engine
}
