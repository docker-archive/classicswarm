package cluster

import (
	"errors"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
)

var (
	ErrNodeNotConnected      = errors.New("node is not connected to docker's REST API")
	ErrNodeAlreadyRegistered = errors.New("node was already added to the cluster")
)

type Cluster struct {
	sync.RWMutex
	eventHandlers []EventHandler
	nodes         map[string]*Node
}

func NewCluster() *Cluster {
	return &Cluster{
		nodes: make(map[string]*Node),
	}
}

func (c *Cluster) Handle(e *Event) error {
	for _, eventHandler := range c.eventHandlers {
		if err := eventHandler.Handle(e); err != nil {
			log.Error(err)
		}
	}
	return nil
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
	return n.Events(c)
}

// Containers returns all the containers in the cluster.
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

// Container returns the container with ID in the cluster
func (c *Cluster) Container(IdOrName string) *Container {
	// Abort immediately if the name is empty.
	if len(IdOrName) == 0 {
		return nil
	}
	for _, container := range c.Containers() {
		// Match ID prefix.
		if strings.HasPrefix(container.Id, IdOrName) {
			return container
		}

		// Match name, /name or engine/name.
		for _, name := range container.Names {
			if name == IdOrName || name == "/"+IdOrName || container.node.ID+name == IdOrName || container.node.Name+name == IdOrName {
				return container
			}
		}
	}

	return nil
}

// Nodes returns the list of nodes in the cluster
func (c *Cluster) Nodes() []*Node {
	nodes := []*Node{}
	c.RLock()
	for _, node := range c.nodes {
		nodes = append(nodes, node)
	}
	c.RUnlock()
	return nodes
}

func (c *Cluster) Node(addr string) *Node {
	for _, node := range c.nodes {
		if node.Addr == addr {
			return node
		}
	}
	return nil
}

func (c *Cluster) Events(h EventHandler) error {
	c.eventHandlers = append(c.eventHandlers, h)
	return nil
}
