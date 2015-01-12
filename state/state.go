package state

import (
	"github.com/samalba/dockerclient"
)

type RequestedState struct {
	Name   string
	Config *dockerclient.ContainerConfig
}
