package state

import (
	"github.com/samalba/dockerclient"
)

// RequestedState is exported
type RequestedState struct {
	ID     string
	Name   string
	Config *dockerclient.ContainerConfig
}
