package cluster

import (
	"crypto/tls"
	"errors"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/state"
	"github.com/samalba/dockerclient"
)

var (
	ErrNodeNotConnected      = errors.New("node is not connected to docker's REST API")
	ErrNodeAlreadyRegistered = errors.New("node was already added to the cluster")
)

type Cluster struct {
	sync.RWMutex
	store           *state.Store
	TLSConfig       *tls.Config
	eventHandlers   []EventHandler
	nodes           map[string]*Node
	OvercommitRatio float64
}

func NewCluster(store *state.Store, tlsConfig *tls.Config, overcommitRatio float64) *Cluster {
	return &Cluster{
		TLSConfig:       tlsConfig,
		nodes:           make(map[string]*Node),
		store:           store,
		OvercommitRatio: overcommitRatio,
	}
}

// Deploy a container into a `specific` node on the cluster.
func (c *Cluster) DeployContainer(node *Node, config *dockerclient.ContainerConfig, name string) (*Container, error) {
	container, err := node.Create(config, name, true)
	if err != nil {
		return nil, err
	}

	// Commit the requested state.
	st := &state.RequestedState{
		ID:     container.Id,
		Name:   name,
		Config: config,
	}
	if err := c.store.Add(container.Id, st); err != nil {
		return nil, err
	}
	return container, nil
}

// Destroys a given `container` from the cluster.
func (c *Cluster) DestroyContainer(container *Container, force bool) error {
	if err := container.Node.Destroy(container, force); err != nil {
		return err
	}
	if err := c.store.Remove(container.Id); err != nil {
		if err == state.ErrNotFound {
			log.Debugf("Container %s not found in the store", container.Id)
			return nil
		}
		return err
	}
	return nil
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
func (c *Cluster) Containers() []*Container {
	c.RLock()
	defer c.RUnlock()

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
func (c *Cluster) Container(IdOrName string) *Container {
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
