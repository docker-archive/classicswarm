package cluster

import "github.com/samalba/dockerclient"

type Container struct {
	VirtualId string
	dockerclient.Container

	Info dockerclient.ContainerInfo
	Node *Node
}
