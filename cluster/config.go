package cluster

import "github.com/samalba/dockerclient"

// ContainerConfig is exported
type ContainerConfig struct {
	dockerclient.ContainerConfig
}
