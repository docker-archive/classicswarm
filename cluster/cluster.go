package cluster

import (
	"github.com/docker/swarm/discovery"
	"github.com/samalba/dockerclient"
)

type Cluster interface {
	CreateContainer(config *dockerclient.ContainerConfig, name string) (*Container, error)
	RemoveContainer(container *Container, force bool) error

	Events(eventsHandler EventHandler)
	Nodes() []*Node
	Containers() []*Container
	Container(IdOrName string) *Container
	NewEntries(entries []*discovery.Entry)
}
