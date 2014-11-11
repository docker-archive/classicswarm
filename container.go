package libcluster

import "github.com/samalba/dockerclient"

type Container struct {
	dockerclient.Container

	node *Node
}

func (c *Container) Start() error {
	return c.node.client.StartContainer(c.Id, nil)
}

func (c *Container) Kill(sig int) error {
	return c.node.client.KillContainer(c.Id)
}

func (c *Container) Stop() error {
	return c.node.client.StopContainer(c.Id, 8)
}

func (c *Container) Restart(timeout int) error {
	return c.node.client.RestartContainer(c.Id, timeout)
}
