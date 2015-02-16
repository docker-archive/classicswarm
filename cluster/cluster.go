package cluster

import "github.com/samalba/dockerclient"

type Cluster interface {
	CreateContainer(config *dockerclient.ContainerConfig, name string) (*Container, error)
	RemoveContainer(container *Container, force bool) error

	Nodes() []*Node
	Containers() []*Container
	Container(IdOrName string) *Container
}
