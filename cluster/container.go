package cluster

import "github.com/samalba/dockerclient"

type Container struct {
	dockerclient.Container

	Info dockerclient.ContainerInfo
	node *Node
}

func (c *Container) Node() *Node {
	return c.node
}
