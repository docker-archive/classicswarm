package node

import (
	"container/list"
	"errors"
	"strings"

	"github.com/docker/swarm/cluster"
)

// Node is an abstract type used by the scheduler
type Node struct {
	ID         string
	IP         string
	Addr       string
	Name       string
	Cpus       int64
	Memory     int64
	Labels     map[string]string
	Containers []*cluster.Container
	Images     []*cluster.Image

	UsedMemory  int64
	UsedCpus    int64
	TotalMemory int64
	TotalCpus   int64

	IsHealthy     bool
	ScheduleQueue *list.List
}

// NewNode creates a node from an engine
func NewNode(e *cluster.Engine) *Node {
	return &Node{
		ID:            e.ID,
		IP:            e.IP,
		Addr:          e.Addr,
		Name:          e.Name,
		Cpus:          e.Cpus,
		Labels:        e.Labels,
		Containers:    e.Containers(),
		Images:        e.Images(),
		UsedMemory:    e.UsedMemory(),
		UsedCpus:      e.UsedCpus(),
		TotalMemory:   e.TotalMemory(),
		TotalCpus:     e.TotalCpus(),
		IsHealthy:     e.IsHealthy(),
		ScheduleQueue: e.ScheduleQueue,
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

//ScheduledList returns a list of scheduled items but not created according to the query
func (n *Node) ScheduledList(query string) []string {
	if n.ScheduleQueue == nil {
		return nil
	}
	candidate := make([]string, 1)
	if query == "container" {
		for e := n.ScheduleQueue.Back(); e != nil; e = e.Prev() {
			candidate = append(candidate, e.Value.(cluster.ScheduledItem).Container)
		}
		return candidate
	}
	if query == "image" {
		for e := n.ScheduleQueue.Back(); e != nil; e = e.Prev() {
			candidate = append(candidate, e.Value.(cluster.ScheduledItem).Image)
		}
		return candidate
	}
	return nil
}

// AddContainer inject a container into the internal state.
func (n *Node) AddContainer(container *cluster.Container) error {
	if container.Info.Config != nil {
		memory := container.Info.Config.Memory
		cpus := container.Info.Config.CpuShares
		if n.TotalMemory-memory < 0 || n.TotalCpus-cpus < 0 {
			return errors.New("not enough resources")
		}
		n.UsedMemory = n.UsedMemory + memory
		n.UsedCpus = n.UsedCpus + cpus
	}
	n.Containers = append(n.Containers, container)
	return nil
}
