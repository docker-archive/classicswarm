package cluster

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/samalba/dockerclient"
)

const (
	// Force-refresh the state of the node this often.
	stateRefreshPeriod = 30 * time.Second
)

func NewNode(id string, addr string) *Node {
	e := &Node{
		ID:         id,
		Addr:       addr,
		Labels:     make(map[string]string),
		ch:         make(chan bool),
		containers: make(map[string]*Container),
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
	if err := n.refreshContainers(); err != nil {
		n.client = nil
		return err
	}

	// Start the update loop.
	go n.refreshLoop()

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
		"storagedriver":   info.Driver,
		"executiondriver": info.ExecutionDriver,
		"kernelversion":   info.KernelVersion,
		"operatingsystem": info.OperatingSystem,
	}
	return nil
}

// Refresh the list and status of containers running on the node.
func (n *Node) refreshContainers() error {
	containers, err := n.client.ListContainers(true, false, "")
	if err != nil {
		return err
	}

	n.Lock()
	defer n.Unlock()

	merged := make(map[string]*Container)
	for _, c := range containers {
		merged, err = n.updateContainer(c, merged)
		if err != nil {
			log.Errorf("[%s] Unable to update state of %s", n.ID, c.Id)
		}
	}

	n.containers = merged

	log.Debugf("[%s] Updated state", n.ID)
	return nil
}

// Refresh the status of a container running on the node.
func (n *Node) refreshContainer(ID string) error {
	containers, err := n.client.ListContainers(true, false, fmt.Sprintf("{%q:[%q]}", "id", ID))
	if err != nil {
		return err
	}

	if len(containers) > 1 {
		// We expect one container, if we get more than one, trigger a full refresh.
		return n.refreshContainers()
	}

	n.Lock()
	defer n.Unlock()

	if len(containers) == 0 {
		// The container doesn't exist on the node, remove it.
		delete(n.containers, ID)
		return nil
	}

	_, err = n.updateContainer(containers[0], n.containers)
	return err
}

func (n *Node) updateContainer(c dockerclient.Container, containers map[string]*Container) (map[string]*Container, error) {
	if current, exists := n.containers[c.Id]; exists {
		// The container exists. Update its state.
		current.Container = c
		containers[current.Id] = current
	} else {
		// This is a brand new container.
		container := &Container{}
		container.Container = c
		container.node = n

		info, err := n.client.InspectContainer(c.Id)
		if err != nil {
			return containers, err
		}
		container.Info = *info

		// real CpuShares -> nb of CPUs
		container.Info.Config.CpuShares = container.Info.Config.CpuShares / 100.0 * n.Cpus

		containers[container.Id] = container
	}
	return containers, nil
}

func (n *Node) refreshContainersAsync() {
	n.ch <- true
}

func (n *Node) refreshLoop() {
	for {
		var err error
		select {
		case <-n.ch:
			err = n.refreshContainers()
		case <-time.After(stateRefreshPeriod):
			err = n.refreshContainers()
		}
		if err != nil {
			log.Errorf("[%s] Updated state failed: %v", n.ID, err)
		}
	}
}

// Return the sum of memory reserved by containers.
func (n *Node) ReservedMemory() int64 {
	var r int64 = 0
	for _, c := range n.containers {
		r += int64(c.Info.Config.Memory)
	}
	return r
}

// Return the memory availalble on this node.
func (n *Node) AvailableMemory() int64 {
	return n.Memory - n.ReservedMemory()
}

// Return the sum of CPUs reserved by containers.
func (n *Node) ReservedCpus() int64 {
	var r int64 = 0
	for _, c := range n.containers {
		r += int64(c.Info.Config.CpuShares)
	}
	return r
}

func (n *Node) AvailalbleCpus() int64 {
	return int64(n.Cpus) - n.ReservedCpus()
}

func (n *Node) Create(config *dockerclient.ContainerConfig, name string, pullImage bool) (*Container, error) {
	var (
		err    error
		id     string
		client = n.client
	)

	newConfig := *config

	// nb of CPUs -> real CpuShares
	newConfig.CpuShares = config.CpuShares * 100 / n.Cpus

	if id, err = client.CreateContainer(&newConfig, name); err != nil {
		// If the error is other than not found, abort immediately.
		if err != dockerclient.ErrNotFound {
			return nil, err
		}
		// Otherwise, try to pull the image...
		if err = n.Pull(config.Image); err != nil {
			return nil, err
		}
		// ...And try again.
		if id, err = client.CreateContainer(&newConfig, name); err != nil {
			return nil, err
		}
	}

	// Register the container immediately while waiting for a state refresh.
	// Force a state refresh to pick up the newly created container.
	n.refreshContainer(id)

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

func (n *Node) Pull(image string) error {
	if err := n.client.PullImage(image); err != nil {
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
	n.refreshContainer(ev.Id)

	// If there is no event handler registered, abort right now.
	if n.eventHandler == nil {
		return
	}

	event := &Event{
		Node: n,
		Type: ev.Status,
		Time: time.Unix(int64(ev.Time), 0),
	}

	if container, ok := n.containers[ev.Id]; ok {
		event.Container = container
	} else {
		event.Container = &Container{
			node: n,
			Container: dockerclient.Container{
				Id:    ev.Id,
				Image: ev.From,
			},
		}
	}

	n.eventHandler.Handle(event)
}

// Used only on tests
func (n *Node) AddContainer(container *Container) {
	n.containers[container.Id] = container
}

func (n *Node) CleanupContainers() {
	n.containers = make(map[string]*Container)
}
