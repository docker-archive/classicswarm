package cluster

import "github.com/samalba/dockerclient"

type Container struct {
	dockerclient.Container

	Info dockerclient.ContainerInfo
	Node *Node
}
