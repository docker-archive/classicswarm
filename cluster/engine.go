package cluster

import (
	"crypto/tls"
	"errors"
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

	// Timeout for requests sent out to the node.
	requestTimeout = 10 * time.Second
)

// NewEngine is exported
func NewEngine(addr, name string, overcommitRatio float64) *Engine {
	e := &Engine{
		addr:            addr,
		name:            name,
		labels:          make(map[string]string),
		ch:              make(chan bool),
		containers:      make(map[string]*Container),
		healthy:         true,
		overcommitRatio: int64(overcommitRatio * 100),
	}
	return e
}

// Engine represent a docker engine
type Engine struct {
	sync.RWMutex

	id     string
	ip     string
	addr   string
	name   string
	Cpus   int64
	Memory int64
	labels map[string]string

	ch              chan bool
	containers      map[string]*Container
	images          []*Image
	Client          dockerclient.Client
	eventHandler    EventHandler
	healthy         bool
	overcommitRatio int64
}

// ID of the docker engine
func (n *Engine) ID() string {
	return n.id
}

// IP of the docker engine
func (n *Engine) IP() string {
	return n.ip
}

// Addr (ip:port or hostname:port) of the docker engine
func (n *Engine) Addr() string {
	return n.addr
}

// Name of the docker engine
func (n *Engine) Name() string {
	return n.name
}

// Labels of the docker engine
func (n *Engine) Labels() map[string]string {
	return n.labels
}

// Connect will initialize a connection to the Docker daemon running on the
// host, gather machine specs (memory, cpu, ...) and monitor state changes.
func (n *Engine) Connect(config *tls.Config) error {
	host, _, err := net.SplitHostPort(n.addr)
	if err != nil {
		return err
	}

	addr, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return err
	}
	n.ip = addr.IP.String()

	c, err := dockerclient.NewDockerClientTimeout("tcp://"+n.addr, config, time.Duration(requestTimeout))
	if err != nil {
		return err
	}

	return n.ConnectClient(c)
}

// ConnectClient get specs of the engine (/info) and then refresh containers and images
func (n *Engine) ConnectClient(client dockerclient.Client) error {
	n.Client = client

	// Fetch the engine labels.
	if err := n.updateSpecs(); err != nil {
		n.Client = nil
		return err
	}

	// Force a state update before returning.
	if err := n.refreshContainers(true); err != nil {
		n.Client = nil
		return err
	}

	if err := n.refreshImages(); err != nil {
		n.Client = nil
		return err
	}

	// Start the update loop.
	go n.refreshLoop()

	// Start monitoring events from the node.
	n.Client.StartMonitorEvents(n.handler, nil)
	n.emitEvent("node_connect")

	return nil
}

// IsConnected returns true if the engine is connected to a remote docker API
func (n *Engine) IsConnected() bool {
	return n.Client != nil
}

// IsHealthy returns true if the engine is healthy
func (n *Engine) IsHealthy() bool {
	return n.healthy
}

// Gather node specs (CPU, memory, constraints, ...).
func (n *Engine) updateSpecs() error {
	info, err := n.Client.Info()
	if err != nil {
		return err
	}
	// Older versions of Docker don't expose the ID field and are not supported
	// by Swarm.  Catch the error ASAP and refuse to connect.
	if len(info.ID) == 0 {
		return fmt.Errorf("node %s is running an unsupported version of Docker Engine. Please upgrade.", n.addr)
	}
	n.id = info.ID
	n.name = info.Name
	n.Cpus = info.NCPU
	n.Memory = info.MemTotal
	n.labels = map[string]string{
		"storagedriver":   info.Driver,
		"executiondriver": info.ExecutionDriver,
		"kernelversion":   info.KernelVersion,
		"operatingsystem": info.OperatingSystem,
	}
	for _, label := range info.Labels {
		kv := strings.SplitN(label, "=", 2)
		n.labels[kv[0]] = kv[1]
	}
	return nil
}

// RemoveImage deletes an image from the node.
func (n *Engine) RemoveImage(image *Image) ([]*dockerclient.ImageDelete, error) {
	return n.Client.RemoveImage(image.Id)
}

// Refresh the list of images on the node.
func (n *Engine) refreshImages() error {
	images, err := n.Client.ListImages()
	if err != nil {
		return err
	}
	n.Lock()
	n.images = nil
	for _, image := range images {
		n.images = append(n.images, &Image{Image: *image, Node: n})
	}
	n.Unlock()
	return nil
}

// Refresh the list and status of containers running on the node. If `full` is
// true, each container will be inspected.
func (n *Engine) refreshContainers(full bool) error {
	containers, err := n.Client.ListContainers(true, false, "")
	if err != nil {
		return err
	}

	merged := make(map[string]*Container)
	for _, c := range containers {
		merged, err = n.updateContainer(c, merged, full)
		if err != nil {
			log.WithFields(log.Fields{"name": n.name, "id": n.id}).Errorf("Unable to update state of container %q", c.Id)
		}
	}

	n.Lock()
	defer n.Unlock()
	n.containers = merged

	log.WithFields(log.Fields{"id": n.id, "name": n.name}).Debugf("Updated node state")
	return nil
}

// RefreshContainer refresh the status of a container running on the node. If `full` is true,
// the container will be inspected.
func (n *Engine) RefreshContainer(ID string, full bool) error {
	containers, err := n.Client.ListContainers(true, false, fmt.Sprintf("{%q:[%q]}", "id", ID))
	if err != nil {
		return err
	}

	if len(containers) > 1 {
		// We expect one container, if we get more than one, trigger a full refresh.
		return n.refreshContainers(full)
	}

	if len(containers) == 0 {
		// The container doesn't exist on the node, remove it.
		n.Lock()
		delete(n.containers, ID)
		n.Unlock()

		return nil
	}

	_, err = n.updateContainer(containers[0], n.containers, full)
	return err
}

func (n *Engine) updateContainer(c dockerclient.Container, containers map[string]*Container, full bool) (map[string]*Container, error) {
	var container *Container

	n.Lock()

	if current, exists := n.containers[c.Id]; exists {
		// The container is already known.
		container = current
	} else {
		// This is a brand new container. We need to do a full refresh.
		container = &Container{
			Node: n,
		}
		full = true
	}

	// Update its internal state.
	container.Container = c
	containers[container.Id] = container

	// Release the lock here as the next step is slow.
	n.Unlock()

	// Update ContainerInfo.
	if full {
		info, err := n.Client.InspectContainer(c.Id)
		if err != nil {
			return nil, err
		}
		container.Info = *info
		// real CpuShares -> nb of CPUs
		container.Info.Config.CpuShares = container.Info.Config.CpuShares / 100.0 * n.Cpus
	}

	return containers, nil
}

func (n *Engine) refreshContainersAsync() {
	n.ch <- true
}

func (n *Engine) refreshLoop() {
	for {
		var err error
		select {
		case <-n.ch:
			err = n.refreshContainers(false)
		case <-time.After(stateRefreshPeriod):
			err = n.refreshContainers(false)
		}

		if err == nil {
			err = n.refreshImages()
		}

		if err != nil {
			if n.healthy {
				n.emitEvent("node_disconnect")
			}
			n.healthy = false
			log.WithFields(log.Fields{"name": n.name, "id": n.id}).Errorf("Flagging node as dead. Updated state failed: %v", err)
		} else {
			if !n.healthy {
				log.WithFields(log.Fields{"name": n.name, "id": n.id}).Info("Node came back to life. Hooray!")
				n.Client.StopAllMonitorEvents()
				n.Client.StartMonitorEvents(n.handler, nil)
				n.emitEvent("node_reconnect")
				if err := n.updateSpecs(); err != nil {
					log.WithFields(log.Fields{"name": n.name, "id": n.id}).Errorf("Update node specs failed: %v", err)
				}
			}
			n.healthy = true
		}
	}
}

func (n *Engine) emitEvent(event string) {
	// If there is no event handler registered, abort right now.
	if n.eventHandler == nil {
		return
	}
	ev := &Event{
		Event: dockerclient.Event{
			Status: event,
			From:   "swarm",
			Time:   time.Now().Unix(),
		},
		Node: n,
	}
	n.eventHandler.Handle(ev)
}

// UsedMemory returns the sum of memory reserved by containers.
func (n *Engine) UsedMemory() int64 {
	var r int64
	n.RLock()
	for _, c := range n.containers {
		r += c.Info.Config.Memory
	}
	n.RUnlock()
	return r
}

// UsedCpus returns the sum of CPUs reserved by containers.
func (n *Engine) UsedCpus() int64 {
	var r int64
	n.RLock()
	for _, c := range n.containers {
		r += c.Info.Config.CpuShares
	}
	n.RUnlock()
	return r
}

// TotalMemory return the engine total memory + overcommit
func (n *Engine) TotalMemory() int64 {
	return n.Memory + (n.Memory * n.overcommitRatio / 100)
}

// TotalCpus return the engine total cpus + overcommit
func (n *Engine) TotalCpus() int64 {
	return n.Cpus + (n.Cpus * n.overcommitRatio / 100)
}

// Pull pulls an image on the engine
func (n *Engine) Pull(image string) error {
	if !strings.Contains(image, ":") {
		image = image + ":latest"
	}
	if err := n.Client.PullImage(image, nil); err != nil {
		return err
	}
	return nil
}

// Events registers an event handler.
func (n *Engine) Events(h EventHandler) error {
	if n.eventHandler != nil {
		return errors.New("event handler already set")
	}
	n.eventHandler = h
	return nil
}

// Containers returns all the containers in the node.
func (n *Engine) Containers() []*Container {
	containers := []*Container{}
	n.RLock()
	for _, container := range n.containers {
		containers = append(containers, container)
	}
	n.RUnlock()
	return containers
}

// Container returns the container with IdOrName in the node.
func (n *Engine) Container(IdOrName string) *Container {
	// Abort immediately if the name is empty.
	if len(IdOrName) == 0 {
		return nil
	}

	n.RLock()
	defer n.RUnlock()

	for _, container := range n.Containers() {
		// Match ID prefix.
		if strings.HasPrefix(container.Id, IdOrName) {
			return container
		}

		// Match name, /name or engine/name.
		for _, name := range container.Names {
			if name == IdOrName || name == "/"+IdOrName || container.Node.ID()+name == IdOrName || container.Node.Name()+name == IdOrName {
				return container
			}
		}
	}

	return nil
}

// Images returns all the images in this engine
func (n *Engine) Images() []*Image {
	images := []*Image{}
	n.RLock()
	for _, image := range n.images {
		images = append(images, image)
	}
	n.RUnlock()
	return images
}

// Image returns the image with IdOrName in the node
func (n *Engine) Image(IdOrName string) *Image {
	n.RLock()
	defer n.RUnlock()

	size := len(IdOrName)
	for _, image := range n.Images() {
		if image.Id == IdOrName || (size > 2 && strings.HasPrefix(image.Id, IdOrName)) {
			return image
		}
		for _, t := range image.RepoTags {
			if t == IdOrName || (size > 2 && strings.HasPrefix(t, IdOrName)) {
				return image
			}
		}
	}
	return nil
}

func (n *Engine) String() string {
	return fmt.Sprintf("node %s addr %s", n.id, n.addr)
}

func (n *Engine) handler(ev *dockerclient.Event, _ chan error, args ...interface{}) {
	// Something changed - refresh our internal state.
	switch ev.Status {
	case "pull", "untag", "delete":
		// These events refer to images so there's no need to update
		// containers.
		n.refreshImages()
	case "start", "die":
		// If the container is started or stopped, we have to do an inspect in
		// order to get the new NetworkSettings.
		n.RefreshContainer(ev.Id, true)
	default:
		// Otherwise, do a "soft" refresh of the container.
		n.RefreshContainer(ev.Id, false)
	}

	// If there is no event handler registered, abort right now.
	if n.eventHandler == nil {
		return
	}

	event := &Event{
		Node:  n,
		Event: *ev,
	}

	n.eventHandler.Handle(event)
}

// AddContainer injects a container into the internal state.
func (n *Engine) AddContainer(container *Container) error {
	n.Lock()
	defer n.Unlock()

	if _, ok := n.containers[container.Id]; ok {
		return errors.New("container already exists")
	}
	n.containers[container.Id] = container
	return nil
}

// Inject an image into the internal state.
func (n *Engine) addImage(image *Image) {
	n.Lock()
	defer n.Unlock()

	n.images = append(n.images, image)
}

// RemoveContainer removes a container from the internal test.
func (n *Engine) RemoveContainer(container *Container) error {
	n.Lock()
	defer n.Unlock()

	if _, ok := n.containers[container.Id]; !ok {
		return errors.New("container not found")
	}
	delete(n.containers, container.Id)
	return nil
}

// Wipes the internal container state.
func (n *Engine) cleanupContainers() {
	n.Lock()
	n.containers = make(map[string]*Container)
	n.Unlock()
}
