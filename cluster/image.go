package cluster

import "github.com/samalba/dockerclient"

type Image struct {
	dockerclient.Image

	Node *Node
}
