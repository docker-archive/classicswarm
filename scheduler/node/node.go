package node

import (
	"errors"
	"strings"
	"sync"

	"github.com/docker/swarm/cluster"
)

var (
	// ErrNotEnoughResources is thrown when a Node does not have any CPU or Memory left
	ErrNotEnoughResources = errors.New("No more resources available on this node")
)

// Node is an abstract type used by the scheduler.
type Node struct {
	sync.RWMutex

	ID         string
	IP         string
	Addr       string
	Name       string
	Labels     map[string]string
	Containers []*cluster.Container
	Images     []*cluster.Image

	UsedMemory  int64
	UsedCpus    int64
	TotalMemory int64
	TotalCpus   int64

	IsHealthy bool
}

// NewNode creates a node from an engine.
func NewNode(e *cluster.Engine) *Node {
	return &Node{
		ID:          e.ID,
		IP:          e.IP,
		Addr:        e.Addr,
		Name:        e.Name,
		Labels:      e.Labels,
		Containers:  e.Containers(),
		Images:      e.Images(true),
		UsedMemory:  e.UsedMemory(),
		UsedCpus:    e.UsedCpus(),
		TotalMemory: e.TotalMemory(),
		TotalCpus:   e.TotalCpus(),
		IsHealthy:   e.IsHealthy(),
	}
}

// Container returns the container with IDOrName in the engine.
func (n *Node) Container(IDOrName string) *cluster.Container {
	// Abort immediately if the name is empty.
	if len(IDOrName) == 0 {
		return nil
	}

	for _, container := range n.Containers {
		// Match ID prefix.
		if strings.HasPrefix(container.Id, IDOrName) {
			return container
		}

		// Match name, /name or engine/name.
		for _, name := range container.Names {
			if name == IDOrName || name == "/"+IDOrName || container.Engine.ID+name == IDOrName || container.Engine.Name+name == IDOrName {
				return container
			}
		}
	}

	return nil
}

// AddContainer injects a container into the internal state.
func (n *Node) AddContainer(container *cluster.Container) error {
	if container.Config != nil {
		memory := container.Config.Memory
		cpus := container.Config.CpuShares
		if n.TotalMemory-memory < 0 || n.TotalCpus-cpus < 0 {
			return ErrNotEnoughResources
		}
		n.UsedMemory = n.UsedMemory + memory
		n.UsedCpus = n.UsedCpus + cpus
	}
	n.Containers = append(n.Containers, container)
	return nil
}

// ReserveResource reserve some resources for a container
func (n *Node) ReserveResource(config *cluster.ContainerConfig) error {
	n.Lock()
	defer n.Unlock()
	if config != nil {
		memory := config.Memory
		cpus := config.CpuShares
		if (n.TotalMemory-n.UsedMemory)-memory < 0 || (n.TotalCpus-n.UsedCpus)-cpus < 0 {
			return ErrNotEnoughResources
		}
		n.UsedMemory = n.UsedMemory + memory
		n.UsedCpus = n.UsedCpus + cpus
	}
	return nil
}

// ReleaseResource releases resource on the node
func (n *Node) ReleaseResource(config *cluster.ContainerConfig) error {
	n.Lock()
	defer n.Unlock()
	if config != nil {
		memory := config.Memory
		cpus := config.CpuShares
		n.UsedMemory = n.UsedMemory - memory
		n.UsedCpus = n.UsedCpus - cpus
	}
	return nil
}
