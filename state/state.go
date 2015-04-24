package state

import "github.com/docker/swarm/cluster"

// RequestedState is exported
type RequestedState struct {
	ID     string
	Name   string
	Config *cluster.ContainerConfig
}
