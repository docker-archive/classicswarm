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
func NewEngine(addr string, overcommitRatio float64) *Engine {
	e := &Engine{
		Addr:            addr,
		Labels:          make(map[string]string),
		ch:              make(chan bool),
		containers:      make(map[string]*Container),
		healthy:         true,
		overcommitRatio: int64(overcommitRatio * 100),
	}
	return e
}

// Engine represents a docker engine
type Engine struct {
	sync.RWMutex

	ID     string
	IP     string
	Addr   string
	Name   string
	Cpus   int64
	Memory int64
	Labels map[string]string

	ch              chan bool
	containers      map[string]*Container
	images          []*Image
	client          dockerclient.Client
	eventHandler    EventHandler
	healthy         bool
	overcommitRatio int64
}

// Connect will initialize a connection to the Docker daemon running on the
// host, gather machine specs (memory, cpu, ...) and monitor state changes.
func (n *Engine) Connect(config *tls.Config) error {
	host, _, err := net.SplitHostPort(n.Addr)
	if err != nil {
		return err
	}

	addr, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return err
	}
	n.IP = addr.IP.String()

	c, err := dockerclient.NewDockerClientTimeout("tcp://"+n.Addr, config, time.Duration(requestTimeout))
	if err != nil {
		return err
	}

	return n.connectClient(c)
}

func (n *Engine) connectClient(client dockerclient.Client) error {
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

	// Start monitoring events from the engine.
	n.client.StartMonitorEvents(n.handler, nil)
	n.emitEvent("engine_connect")

	return nil
}

// isConnected returns true if the engine is connected to a remote docker API
func (n *Engine) isConnected() bool {
	return n.client != nil
}

// IsHealthy returns true if the engine is healthy
func (n *Engine) IsHealthy() bool {
	return n.healthy
}

// Gather engine specs (CPU, memory, constraints, ...).
func (n *Engine) updateSpecs() error {
	info, err := n.client.Info()
	if err != nil {
		return err
	}

	if info.NCPU == 0 || info.MemTotal == 0 {
		return fmt.Errorf("cannot get resources for this engine, make sure %s is a Docker Engine, not a Swarm manager", n.Addr)
	}

	// Older versions of Docker don't expose the ID field and are not supported
	// by Swarm.  Catch the error ASAP and refuse to connect.
	if len(info.ID) == 0 {
		return fmt.Errorf("engine %s is running an unsupported version of Docker Engine. Please upgrade", n.Addr)
	}
	n.ID = info.ID
	n.Name = info.Name
	n.Cpus = info.NCPU
	n.Memory = info.MemTotal
	n.Labels = map[string]string{
		"storagedriver":   info.Driver,
		"executiondriver": info.ExecutionDriver,
		"kernelversion":   info.KernelVersion,
		"operatingsystem": info.OperatingSystem,
	}
	for _, label := range info.Labels {
		kv := strings.SplitN(label, "=", 2)
		n.Labels[kv[0]] = kv[1]
	}
	return nil
}

// RemoveImage deletes an image from the engine.
func (n *Engine) RemoveImage(image *Image) ([]*dockerclient.ImageDelete, error) {
	return n.client.RemoveImage(image.Id)
}

// Refresh the list of images on the engine.
func (n *Engine) refreshImages() error {
	images, err := n.client.ListImages()
	if err != nil {
		return err
	}
	n.Lock()
	n.images = nil
	for _, image := range images {
		n.images = append(n.images, &Image{Image: *image, Engine: n})
	}
	n.Unlock()
	return nil
}

// Refresh the list and status of containers running on the engine. If `full` is
// true, each container will be inspected.
func (n *Engine) refreshContainers(full bool) error {
	containers, err := n.client.ListContainers(true, false, "")
	if err != nil {
		return err
	}

	merged := make(map[string]*Container)
	for _, c := range containers {
		merged, err = n.updateContainer(c, merged, full)
		if err != nil {
			log.WithFields(log.Fields{"name": n.Name, "id": n.ID}).Errorf("Unable to update state of container %q", c.Id)
		}
	}

	n.Lock()
	defer n.Unlock()
	n.containers = merged

	log.WithFields(log.Fields{"id": n.ID, "name": n.Name}).Debugf("Updated engine state")
	return nil
}

// Refresh the status of a container running on the engine. If `full` is true,
// the container will be inspected.
func (n *Engine) refreshContainer(ID string, full bool) error {
	containers, err := n.client.ListContainers(true, false, fmt.Sprintf("{%q:[%q]}", "id", ID))
	if err != nil {
		return err
	}

	if len(containers) > 1 {
		// We expect one container, if we get more than one, trigger a full refresh.
		return n.refreshContainers(full)
	}

	if len(containers) == 0 {
		// The container doesn't exist on the engine, remove it.
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

	n.RLock()
	if current, exists := n.containers[c.Id]; exists {
		// The container is already known.
		container = current
	} else {
		// This is a brand new container. We need to do a full refresh.
		container = &Container{
			Engine: n,
		}
		full = true
	}
	// Release the lock here as the next step is slow.
	// Trade-off: If updateContainer() is called concurrently for the same
	// container, we will end up doing a full refresh twice and the original
	// container (containers[container.Id]) will get replaced.
	n.RUnlock()

	// Update ContainerInfo.
	if full {
		info, err := n.client.InspectContainer(c.Id)
		if err != nil {
			return nil, err
		}
		container.Info = *info
		// real CpuShares -> nb of CPUs
		container.Info.Config.CpuShares = container.Info.Config.CpuShares * 1024.0 / n.Cpus
	}

	// Update its internal state.
	n.Lock()
	container.Container = c
	containers[container.Id] = container
	n.Unlock()

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
				n.emitEvent("engine_disconnect")
			}
			n.healthy = false
			log.WithFields(log.Fields{"name": n.Name, "id": n.ID}).Errorf("Flagging engine as dead. Updated state failed: %v", err)
		} else {
			if !n.healthy {
				log.WithFields(log.Fields{"name": n.Name, "id": n.ID}).Info("Engine came back to life. Hooray!")
				n.client.StopAllMonitorEvents()
				n.client.StartMonitorEvents(n.handler, nil)
				n.emitEvent("engine_reconnect")
				if err := n.updateSpecs(); err != nil {
					log.WithFields(log.Fields{"name": n.Name, "id": n.ID}).Errorf("Update engine specs failed: %v", err)
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
		Engine: n,
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

// TotalMemory returns the total memory + overcommit
func (n *Engine) TotalMemory() int64 {
	return n.Memory + (n.Memory * n.overcommitRatio / 100)
}

// TotalCpus returns the total cpus + overcommit
func (n *Engine) TotalCpus() int64 {
	return n.Cpus + (n.Cpus * n.overcommitRatio / 100)
}

// Create a new container
func (n *Engine) Create(config *dockerclient.ContainerConfig, name string, pullImage bool) (*Container, error) {
	var (
		err    error
		id     string
		client = n.client
	)

	newConfig := *config

	// nb of CPUs -> real CpuShares
	newConfig.CpuShares = config.CpuShares * 1024 / n.Cpus

	if id, err = client.CreateContainer(&newConfig, name); err != nil {
		// If the error is other than not found, abort immediately.
		if err != dockerclient.ErrNotFound || !pullImage {
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
	n.refreshContainer(id, true)

	n.RLock()
	defer n.RUnlock()

	return n.containers[id], nil
}

// Destroy and remove a container from the engine.
func (n *Engine) Destroy(container *Container, force bool) error {
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

// Pull an image on the node
func (n *Engine) Pull(image string) error {
	if !strings.Contains(image, ":") {
		image = image + ":latest"
	}
	if err := n.client.PullImage(image, nil); err != nil {
		return err
	}
	return nil
}

// Events register an event handler.
func (n *Engine) Events(h EventHandler) error {
	if n.eventHandler != nil {
		return errors.New("event handler already set")
	}
	n.eventHandler = h
	return nil
}

// Containers returns all the containers in the engine.
func (n *Engine) Containers() []*Container {
	containers := []*Container{}
	n.RLock()
	for _, container := range n.containers {
		containers = append(containers, container)
	}
	n.RUnlock()
	return containers
}

// Container returns the container with IDOrName in the engine.
func (n *Engine) Container(IDOrName string) *Container {
	// Abort immediately if the name is empty.
	if len(IDOrName) == 0 {
		return nil
	}

	n.RLock()
	defer n.RUnlock()

	for _, container := range n.Containers() {
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

// Images returns all the images in the engine
func (n *Engine) Images() []*Image {
	images := []*Image{}
	n.RLock()

	for _, image := range n.images {
		images = append(images, image)
	}
	n.RUnlock()
	return images
}

// Image returns the image with IDOrName in the engine
func (n *Engine) Image(IDOrName string) *Image {
	n.RLock()
	defer n.RUnlock()

	for _, image := range n.images {
		if image.Match(IDOrName) {
			return image
		}
	}
	return nil
}

func (n *Engine) String() string {
	return fmt.Sprintf("engine %s addr %s", n.ID, n.Addr)
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
		n.refreshContainer(ev.Id, true)
	default:
		// Otherwise, do a "soft" refresh of the container.
		n.refreshContainer(ev.Id, false)
	}

	// If there is no event handler registered, abort right now.
	if n.eventHandler == nil {
		return
	}

	event := &Event{
		Engine: n,
		Event:  *ev,
	}

	n.eventHandler.Handle(event)
}

// AddContainer inject a container into the internal state.
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

// Remove a container from the internal test.
func (n *Engine) removeContainer(container *Container) error {
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
