package libcluster

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/samalba/dockerclient"
)

const (
	// Force-refresh the state of the node this often.
	stateRefreshPeriod = 30 * time.Second
)

func NewNode(id string, addr string) *Node {
	e := &Node{
		ID:     id,
		Addr:   addr,
		Labels: make(map[string]string),
		ch:     make(chan bool),
	}
	return e
}

type Node struct {
	sync.Mutex

	ID     string
	IP     string
	Addr   string
	Cpus   int
	Memory int64
	Labels map[string]string

	ch           chan bool
	containers   map[string]*Container
	client       dockerclient.Client
	eventHandler EventHandler
}

// Connect will initialize a connection to the Docker daemon running on the
// host, gather machine specs (memory, cpu, ...) and monitor state changes.
func (n *Node) Connect(config *tls.Config) error {
	c, err := dockerclient.NewDockerClient(n.Addr, config)
	if err != nil {
		return err
	}

	addr, err := net.ResolveIPAddr("ip4", strings.Split(c.URL.Host, ":")[0])
	if err != nil {
		return err
	}
	n.IP = addr.IP.String()

	return n.connectClient(c)
}

func (n *Node) connectClient(client dockerclient.Client) error {
	n.client = client

	// Fetch the engine labels.
	if err := n.updateSpecs(); err != nil {
		n.client = nil
		return err
	}

	// Force a state update before returning.
	if err := n.updateState(); err != nil {
		n.client = nil
		return err
	}

	// Start the update loop.
	go n.updateLoop()

	// Start monitoring events from the Node.
	n.client.StartMonitorEvents(n.handler)

	return nil
}

// IsConnected returns true if the engine is connected to a remote docker API
func (e *Node) IsConnected() bool {
	return e.client != nil
}

// Gather node specs (CPU, memory, constraints, ...).
func (n *Node) updateSpecs() error {
	info, err := n.client.Info()
	if err != nil {
		return err
	}
	n.Cpus = info.NCPU
	n.Memory = info.MemTotal
	n.Labels = map[string]string{
		"graphdriver":     info.Driver,
		"executiondriver": info.ExecutionDriver,
		"kernelversion":   info.KernelVersion,
		"operatingsystem": info.OperatingSystem,
	}
	return nil
}

// Refresh the list and status of containers running on the node.
func (n *Node) updateState() error {
	containers, err := n.client.ListContainers(true)
	if err != nil {
		return err
	}

	n.Lock()
	defer n.Unlock()

	n.containers = make(map[string]*Container)
	for _, c := range containers {
		container := &Container{}
		container.Container = c
		container.node = n
		n.containers[container.Id] = container
	}

	log.Printf("[%s] Updated state", n.ID)
	return nil
}

func (n *Node) updateStateAsync() {
	n.ch <- true
}

func (n *Node) updateLoop() {
	for {
		var err error
		select {
		case <-n.ch:
			err = n.updateState()
		case <-time.After(stateRefreshPeriod):
			err = n.updateState()
		}
		if err != nil {
			log.Printf("[%s] Updated state failed: %v", n.ID, err)
		}
	}
}

func (n *Node) Create(config *dockerclient.ContainerConfig, name string, pullImage bool) (*Container, error) {
	var (
		err    error
		id     string
		client = n.client
	)

	if id, err = client.CreateContainer(config, name); err != nil {
		// If the error is other than not found, abort immediately.
		if err != dockerclient.ErrNotFound {
			return nil, err
		}
		// Otherwise, try to pull the image...
		if err = n.Pull(config.Image); err != nil {
			return nil, err
		}
		// ...And try again.
		if id, err = client.CreateContainer(config, name); err != nil {
			return nil, err
		}
	}

	// Register the container immediately while waiting for a state refresh.
	// Force a state refresh to pick up the newly created container.
	n.updateState()

	return n.containers[id], nil
}

func (n *Node) ListImages() ([]string, error) {
	images, err := n.client.ListImages()
	if err != nil {
		return nil, err
	}

	out := []string{}

	for _, i := range images {
		for _, t := range i.RepoTags {
			out = append(out, t)
		}
	}

	return out, nil
}

func (n *Node) Remove(container *Container, force bool) error {
	if err := n.client.RemoveContainer(container.Id, force); err != nil {
		return err
	}

	// Remove the container from the state. Eventually, the state refresh loop
	// will rewrite this.
	n.Lock()
	defer n.Unlock()
	delete(n.containers, container.Id)

	return nil
}

func (e *Node) Pull(image string) error {
	if err := e.client.PullImage(image); err != nil {
		return err
	}
	return nil
}

// Register an event handler.
func (n *Node) Events(h EventHandler) error {
	if n.eventHandler != nil {
		return fmt.Errorf("event handler already set")
	}
	n.eventHandler = h
	return nil
}

func (n *Node) Containers() map[string]*Container {
	return n.containers
}

func (n *Node) String() string {
	return fmt.Sprintf("node %s addr %s", n.ID, n.Addr)
}

func (n *Node) handler(ev *dockerclient.Event, args ...interface{}) {
	// Something changed - refresh our internal state.
	n.updateState()

	// If there is no event handler registered, abort right now.
	if n.eventHandler == nil {
		return
	}

	event := &Event{
		Node:      n,
		Type:      ev.Status,
		Time:      time.Unix(int64(ev.Time), 0),
		Container: n.containers[ev.Id],
	}

	n.eventHandler.Handle(event)
}
