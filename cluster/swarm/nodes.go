package swarm

import (
	"errors"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
)

var (
	ErrNodeNotConnected      = errors.New("node is not connected to docker's REST API")
	ErrNodeAlreadyRegistered = errors.New("node was already")
)

type Nodes struct {
	sync.RWMutex

	eventHandlers []cluster.EventHandler
	nodes         map[string]*cluster.Node
}

func NewNodes() *Nodes {
	return &Nodes{
		nodes: make(map[string]*cluster.Node),
	}
}

func (c *Nodes) Handle(e *cluster.Event) error {
	for _, eventHandler := range c.eventHandlers {
		if err := eventHandler.Handle(e); err != nil {
			log.Error(err)
		}
	}
	return nil
}

// Register a node within the cluster. The node must have been already
// initialized.
func (c *Nodes) Add(n *cluster.Node) error {
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

// Containers returns all the images in the cluster.
func (c *Nodes) Images() []*cluster.Image {
	c.Lock()
	defer c.Unlock()

	out := []*cluster.Image{}
	for _, n := range c.nodes {
		out = append(out, n.Images()...)
	}

	return out
}

// Image returns an image with IdOrName in the cluster
func (c *Nodes) Image(IdOrName string) *cluster.Image {
	// Abort immediately if the name is empty.
	if len(IdOrName) == 0 {
		return nil
	}

	c.RLock()
	defer c.RUnlock()
	for _, n := range c.nodes {
		if image := n.Image(IdOrName); image != nil {
			return image
		}
	}

	return nil
}

// Containers returns all the containers in the cluster.
func (c *Nodes) Containers() []*cluster.Container {
	c.Lock()
	defer c.Unlock()

	out := []*cluster.Container{}
	for _, n := range c.nodes {
		out = append(out, n.Containers()...)
	}

	return out
}

// Container returns the container with IdOrName in the cluster
func (c *Nodes) Container(IdOrName string) *cluster.Container {
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
func (c *Nodes) List() []*cluster.Node {
	nodes := []*cluster.Node{}
	c.RLock()
	for _, node := range c.nodes {
		nodes = append(nodes, node)
	}
	c.RUnlock()
	return nodes
}

func (c *Nodes) Get(addr string) *cluster.Node {
	for _, node := range c.nodes {
		if node.Addr == addr {
			return node
		}
	}
	return nil
}

func (c *Nodes) Events(h cluster.EventHandler) error {
	c.eventHandlers = append(c.eventHandlers, h)
	return nil
}
