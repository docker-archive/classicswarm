package libcluster

import (
	"errors"
	"sync"
)

var (
	ErrNodeNotConnected      = errors.New("node is not connected to docker's REST API")
	ErrNodeAlreadyRegistered = errors.New("node was already added to the cluster")
)

type Cluster struct {
	sync.Mutex
	nodes map[string]*Node
}

func NewCluster() *Cluster {
	return &Cluster{
		nodes: make(map[string]*Node),
	}
}

// Register a node within the cluster. The node must have been already
// initialized.
func (c *Cluster) AddNode(n *Node) error {
	if !n.IsConnected() {
		return ErrNodeNotConnected
	}

	c.Lock()
	defer c.Unlock()

	if _, exists := c.nodes[n.ID]; exists {
		return ErrNodeAlreadyRegistered
	}

	c.nodes[n.ID] = n
	return nil
}

// Containers returns all the containers running in the cluster.
func (c *Cluster) Containers() []*Container {
	c.Lock()
	defer c.Unlock()

	out := []*Container{}
	for _, n := range c.nodes {
		containers := n.Containers()
		for _, container := range containers {
			out = append(out, container)
		}
	}

	return out
}
