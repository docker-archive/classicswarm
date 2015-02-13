package cluster

import (
	"errors"
	"sync"

	log "github.com/Sirupsen/logrus"
)

var (
	ErrNodeNotConnected      = errors.New("node is not connected to docker's REST API")
	ErrNodeAlreadyRegistered = errors.New("node was already")
)

type Nodes struct {
	sync.RWMutex

	eventHandlers []EventHandler
	nodes         map[string]*Node
}

func NewNodes() *Nodes {
	return &Nodes{
		nodes: make(map[string]*Node),
	}
}

func (c *Nodes) Handle(e *Event) error {
	for _, eventHandler := range c.eventHandlers {
		if err := eventHandler.Handle(e); err != nil {
			log.Error(err)
		}
	}
	return nil
}

// Register a node within the cluster. The node must have been already
// initialized.
func (c *Nodes) Add(n *Node) error {
	if !n.IsConnected() {
		return ErrNodeNotConnected
	}

	c.Lock()
	defer c.Unlock()

	if old, exists := c.nodes[n.ID]; exists {
		if old.IP != n.IP {
			log.Errorf("ID duplicated. %s shared by %s and %s", n.ID, old.IP, n.IP)
		}
		return ErrNodeAlreadyRegistered
	}

	c.nodes[n.ID] = n
	return n.Events(c)
}

// Containers returns all the containers in the cluster.
func (c *Nodes) Containers() []*Container {
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

// Container returns the container with IdOrName in the cluster
func (c *Nodes) Container(IdOrName string) *Container {
	// Abort immediately if the name is empty.
	if len(IdOrName) == 0 {
		return nil
	}

	c.RLock()
	defer c.RUnlock()
	for _, n := range c.nodes {
		if container := n.Container(IdOrName); container != nil {
			return container
		}
	}

	return nil
}

// Nodes returns the list of nodes in the cluster
func (c *Nodes) List() []*Node {
	nodes := []*Node{}
	c.RLock()
	for _, node := range c.nodes {
		nodes = append(nodes, node)
	}
	c.RUnlock()
	return nodes
}

func (c *Nodes) Get(addr string) *Node {
	for _, node := range c.nodes {
		if node.Addr == addr {
			return node
		}
	}
	return nil
}

func (c *Nodes) Events(h EventHandler) error {
	c.eventHandlers = append(c.eventHandlers, h)
	return nil
}
