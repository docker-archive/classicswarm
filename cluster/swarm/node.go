package swarm

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

const (
	// Force-refresh the state of the node this often.
	stateRefreshPeriod = 30 * time.Second

	// Timeout for requests sent out to the node.
	requestTimeout = 10 * time.Second
)

func NewNode(addr string, overcommitRatio float64) *node {
	e := &node{
		addr:            addr,
		labels:          make(map[string]string),
		ch:              make(chan bool),
		containers:      make(map[string]*cluster.Container),
		healthy:         true,
		overcommitRatio: int64(overcommitRatio * 100),
	}
	return e
}

type node struct {
	sync.RWMutex

	id     string
	ip     string
	addr   string
	name   string
	Cpus   int64
	Memory int64
	labels map[string]string

	ch              chan bool
	containers      map[string]*cluster.Container
	images          []*cluster.Image
	client          dockerclient.Client
	eventHandler    cluster.EventHandler
	healthy         bool
	overcommitRatio int64
}

func (n *node) ID() string {
	return n.id
}

func (n *node) IP() string {
	return n.ip
}

func (n *node) Addr() string {
	return n.addr
}

func (n *node) Name() string {
	return n.name
}

func (n *node) Labels() map[string]string {
	return n.labels
}

// Connect will initialize a connection to the Docker daemon running on the
// host, gather machine specs (memory, cpu, ...) and monitor state changes.
func (n *node) connect(config *tls.Config) error {
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

	return n.connectClient(c)
}

func (n *node) connectClient(client dockerclient.Client) error {
	n.client = client

	// Fetch the engine labels.
	if err := n.updateSpecs(); err != nil {
		n.client = nil
		return err
	}

	// Force a state update before returning.
	if err := n.refreshContainers(true); err != nil {
		n.client = nil
		return err
	}

	if err := n.refreshImages(); err != nil {
		n.client = nil
		return err
	}

	// Start the update loop.
	go n.refreshLoop()

	// Start monitoring events from the node.
	n.client.StartMonitorEvents(n.handler, nil)
	n.emitEvent("node_connect")

	return nil
}

// isConnected returns true if the engine is connected to a remote docker API
func (n *node) isConnected() bool {
	return n.client != nil
}

func (n *node) IsHealthy() bool {
	return n.healthy
}

// Gather node specs (CPU, memory, constraints, ...).
func (n *node) updateSpecs() error {
	info, err := n.client.Info()
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

// Delete an image from the node.
func (n *node) removeImage(image *cluster.Image) ([]*dockerclient.ImageDelete, error) {
	return n.client.RemoveImage(image.Id)
}

// Refresh the list of images on the node.
func (n *node) refreshImages() error {
	images, err := n.client.ListImages()
	if err != nil {
		return err
	}
	n.Lock()
	n.images = nil
	for _, image := range images {
		n.images = append(n.images, &cluster.Image{Image: *image, Node: n})
	}
	n.Unlock()
	return nil
}

// Refresh the list and status of containers running on the node. If `full` is
// true, each container will be inspected.
func (n *node) refreshContainers(full bool) error {
	containers, err := n.client.ListContainers(true, false, "")
	if err != nil {
		return err
	}

	merged := make(map[string]*cluster.Container)
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

// Refresh the status of a container running on the node. If `full` is true,
// the container will be inspected.
func (n *node) refreshContainer(ID string, full bool) error {
	containers, err := n.client.ListContainers(true, false, fmt.Sprintf("{%q:[%q]}", "id", ID))
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

func (n *node) updateContainer(c dockerclient.Container, containers map[string]*cluster.Container, full bool) (map[string]*cluster.Container, error) {
	var container *cluster.Container

	n.Lock()

	if current, exists := n.containers[c.Id]; exists {
		// The container is already known.
		container = current
	} else {
		// This is a brand new container. We need to do a full refresh.
		container = &cluster.Container{
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
		info, err := n.client.InspectContainer(c.Id)
		if err != nil {
			return nil, err
		}
		container.Info = *info
		// real CpuShares -> nb of CPUs
		container.Info.Config.CpuShares = container.Info.Config.CpuShares / 100.0 * n.Cpus
	}

	return containers, nil
}

func (n *node) refreshContainersAsync() {
	n.ch <- true
}

func (n *node) refreshLoop() {
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
				n.client.StopAllMonitorEvents()
				n.client.StartMonitorEvents(n.handler, nil)
				n.emitEvent("node_reconnect")
				if err := n.updateSpecs(); err != nil {
					log.WithFields(log.Fields{"name": n.name, "id": n.id}).Errorf("Update node specs failed: %v", err)
				}
			}
			n.healthy = true
		}
	}
}

func (n *node) emitEvent(event string) {
	// If there is no event handler registered, abort right now.
	if n.eventHandler == nil {
		return
	}
	ev := &cluster.Event{
		Event: dockerclient.Event{
			Status: event,
			From:   "swarm",
			Time:   time.Now().Unix(),
		},
		Node: n,
	}
	n.eventHandler.Handle(ev)
}

// Return the sum of memory reserved by containers.
func (n *node) UsedMemory() int64 {
	var r int64 = 0
	n.RLock()
	for _, c := range n.containers {
		r += c.Info.Config.Memory
	}
	n.RUnlock()
	return r
}

// Return the sum of CPUs reserved by containers.
func (n *node) UsedCpus() int64 {
	var r int64 = 0
	n.RLock()
	for _, c := range n.containers {
		r += c.Info.Config.CpuShares
	}
	n.RUnlock()
	return r
}

func (n *node) TotalMemory() int64 {
	return n.Memory + (n.Memory * n.overcommitRatio / 100)
}

func (n *node) TotalCpus() int64 {
	return n.Cpus + (n.Cpus * n.overcommitRatio / 100)
}

func (n *node) create(config *dockerclient.ContainerConfig, name string, pullImage bool) (*cluster.Container, error) {
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
		if err != dockerclient.ErrNotFound || !pullImage {
			return nil, err
		}
		// Otherwise, try to pull the image...
		if err = n.pull(config.Image); err != nil {
			return nil, err
		}
		// ...And try again.
		if id, err = client.CreateContainer(&newConfig, name); err != nil {
			return nil, err
		}
	}

	// Register the container immediately while waiting for a state refresh.
	// Force a state refresh to pick up the newly created container.
	n.refreshContainer(id, true)

	n.RLock()
	defer n.RUnlock()

	return n.containers[id], nil
}

// Destroy and remove a container from the node.
func (n *node) destroy(container *cluster.Container, force bool) error {
	if err := n.client.RemoveContainer(container.Id, force, true); err != nil {
		return err
	}

	// Remove the container from the state. Eventually, the state refresh loop
	// will rewrite this.
	n.Lock()
	defer n.Unlock()
	delete(n.containers, container.Id)

	return nil
}

func (n *node) pull(image string) error {
	if !strings.Contains(image, ":") {
		image = image + ":latest"
	}
	if err := n.client.PullImage(image, nil); err != nil {
		return err
	}
	return nil
}

// Register an event handler.
func (n *node) events(h cluster.EventHandler) error {
	if n.eventHandler != nil {
		return errors.New("event handler already set")
	}
	n.eventHandler = h
	return nil
}

// Containers returns all the containers in the node.
func (n *node) Containers() []*cluster.Container {
	containers := []*cluster.Container{}
	n.RLock()
	for _, container := range n.containers {
		containers = append(containers, container)
	}
	n.RUnlock()
	return containers
}

// Container returns the container with IdOrName in the node.
func (n *node) Container(IdOrName string) *cluster.Container {
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

// Images returns all the images in the node
func (n *node) Images() []*cluster.Image {
	images := []*cluster.Image{}
	n.RLock()

	for _, image := range n.images {
		images = append(images, image)
	}
	n.RUnlock()
	return images
}

// Image returns the image with IdOrName in the node
func (n *node) Image(IdOrName string) *cluster.Image {
	n.RLock()
	defer n.RUnlock()

	for _, image := range n.images {
		if image.Match(IdOrName) {
			return image
		}
	}
	return nil
}

func (n *node) String() string {
	return fmt.Sprintf("node %s addr %s", n.id, n.addr)
}

func (n *node) handler(ev *dockerclient.Event, _ chan error, args ...interface{}) {
	// Something changed - refresh our internal state.
	switch ev.Status {
	case "pull", "untag", "delete":
		// These events refer to images so there's no need to update
		// containers.
		n.refreshImages()
	case "start", "die":
		// If the container is started or stopped, we have to do an inspect in
		// order to get the new NetworkSettings.
		n.refreshContainer(ev.Id, true)
	default:
		// Otherwise, do a "soft" refresh of the container.
		n.refreshContainer(ev.Id, false)
	}

	// If there is no event handler registered, abort right now.
	if n.eventHandler == nil {
		return
	}

	event := &cluster.Event{
		Node:  n,
		Event: *ev,
	}

	n.eventHandler.Handle(event)
}

// Inject a container into the internal state.
func (n *node) addContainer(container *cluster.Container) error {
	n.Lock()
	defer n.Unlock()

	if _, ok := n.containers[container.Id]; ok {
		return errors.New("container already exists")
	}
	n.containers[container.Id] = container
	return nil
}

// Inject an image into the internal state.
func (n *node) addImage(image *cluster.Image) {
	n.Lock()
	defer n.Unlock()

	n.images = append(n.images, image)
}

// Remove a container from the internal test.
func (n *node) removeContainer(container *cluster.Container) error {
	n.Lock()
	defer n.Unlock()

	if _, ok := n.containers[container.Id]; !ok {
		return errors.New("container not found")
	}
	delete(n.containers, container.Id)
	return nil
}

// Wipes the internal container state.
func (n *node) cleanupContainers() {
	n.Lock()
	n.containers = make(map[string]*cluster.Container)
	n.Unlock()
}
