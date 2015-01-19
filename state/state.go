package state

import (
	"github.com/samalba/dockerclient"
)

type RequestedState struct {
	ID     string
	Name   string
	Config *dockerclient.ContainerConfig
}
