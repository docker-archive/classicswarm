package cluster

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/version"
	"github.com/samalba/dockerclient"
)

const (
	// Force-refresh the state of the engine this often.
	stateRefreshPeriod = 30 * time.Second

	// Timeout for requests sent out to the engine.
	requestTimeout = 10 * time.Second

	// Minimum docker engine version supported by swarm.
	minSupportedVersion = version.Version("1.6.0")
)

// NewEngine is exported
func NewEngine(addr string, overcommitRatio float64) *Engine {
	e := &Engine{
		Addr:            addr,
		Labels:          make(map[string]string),
		stopCh:          make(chan struct{}),
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

	stopCh          chan struct{}
	containers      map[string]*Container
	images          []*Image
	client          dockerclient.Client
	eventHandler    EventHandler
	healthy         bool
	overcommitRatio int64
}

// Connect will initialize a connection to the Docker daemon running on the
// host, gather machine specs (memory, cpu, ...) and monitor state changes.
func (e *Engine) Connect(config *tls.Config) error {
	host, _, err := net.SplitHostPort(e.Addr)
	if err != nil {
		return err
	}

	addr, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return err
	}
	e.IP = addr.IP.String()

	c, err := dockerclient.NewDockerClientTimeout("tcp://"+e.Addr, config, time.Duration(requestTimeout))
	if err != nil {
		return err
	}

	return e.ConnectWithClient(c)
}

// ConnectWithClient is exported
func (e *Engine) ConnectWithClient(client dockerclient.Client) error {
	e.client = client

	// Fetch the engine labels.
	if err := e.updateSpecs(); err != nil {
		e.client = nil
		return err
	}

	// Force a state update before returning.
	if err := e.RefreshContainers(true); err != nil {
		e.client = nil
		return err
	}

	if err := e.RefreshImages(); err != nil {
		e.client = nil
		return err
	}

	// Start the update loop.
	go e.refreshLoop()

	// Start monitoring events from the engine.
	e.client.StartMonitorEvents(e.handler, nil)
	e.emitEvent("engine_connect")

	return nil
}

// Disconnect will stop all monitoring of the engine.
// The Engine object cannot be further used without reconnecting it first.
func (e *Engine) Disconnect() {
	// do not close the chan, so it wait until the refreshLoop goroutine stops
	e.stopCh <- struct{}{}
	e.client.StopAllMonitorEvents()
	e.client = nil
	e.emitEvent("engine_disconnect")
}

// isConnected returns true if the engine is connected to a remote docker API
func (e *Engine) isConnected() bool {
	return e.client != nil
}

// IsHealthy returns true if the engine is healthy
func (e *Engine) IsHealthy() bool {
	return e.healthy
}

// Gather engine specs (CPU, memory, constraints, ...).
func (e *Engine) updateSpecs() error {
	info, err := e.client.Info()
	if err != nil {
		return err
	}

	if info.NCPU == 0 || info.MemTotal == 0 {
		return fmt.Errorf("cannot get resources for this engine, make sure %s is a Docker Engine, not a Swarm manager", e.Addr)
	}

	v, err := e.client.Version()
	if err != nil {
		return err
	}

	engineVersion := version.Version(v.Version)

	// Older versions of Docker don't expose the ID field, Labels and are not supported
	// by Swarm.  Catch the error ASAP and refuse to connect.
	if engineVersion.LessThan(minSupportedVersion) {
		return fmt.Errorf("engine %s is running an unsupported version of Docker Engine. Please upgrade to at least %s", e.Addr, minSupportedVersion)
	}

	e.ID = info.ID
	e.Name = info.Name
	e.Cpus = info.NCPU
	e.Memory = info.MemTotal
	e.Labels = map[string]string{
		"storagedriver":   info.Driver,
		"executiondriver": info.ExecutionDriver,
		"kernelversion":   info.KernelVersion,
		"operatingsystem": info.OperatingSystem,
	}
	for _, label := range info.Labels {
		kv := strings.SplitN(label, "=", 2)
		e.Labels[kv[0]] = kv[1]
	}
	return nil
}

// RemoveImage deletes an image from the engine.
func (e *Engine) RemoveImage(image *Image, name string) ([]*dockerclient.ImageDelete, error) {
	return e.client.RemoveImage(name)
}

// RefreshImages refreshes the list of images on the engine.
func (e *Engine) RefreshImages() error {
	images, err := e.client.ListImages()
	if err != nil {
		return err
	}
	e.Lock()
	e.images = nil
	for _, image := range images {
		e.images = append(e.images, &Image{Image: *image, Engine: e})
	}
	e.Unlock()
	return nil
}

// RefreshContainers will refresh the list and status of containers running on the engine. If `full` is
// true, each container will be inspected.
// FIXME: unexport this method after mesos scheduler stops using it directly
func (e *Engine) RefreshContainers(full bool) error {
	containers, err := e.client.ListContainers(true, false, "")
	if err != nil {
		return err
	}

	merged := make(map[string]*Container)
	for _, c := range containers {
		merged, err = e.updateContainer(c, merged, full)
		if err != nil {
			log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).Errorf("Unable to update state of container %q", c.Id)
		}
	}

	e.Lock()
	defer e.Unlock()
	e.containers = merged

	log.WithFields(log.Fields{"id": e.ID, "name": e.Name}).Debugf("Updated engine state")
	return nil
}

// Refresh the status of a container running on the engine. If `full` is true,
// the container will be inspected.
func (e *Engine) refreshContainer(ID string, full bool) error {
	containers, err := e.client.ListContainers(true, false, fmt.Sprintf("{%q:[%q]}", "id", ID))
	if err != nil {
		return err
	}

	if len(containers) > 1 {
		// We expect one container, if we get more than one, trigger a full refresh.
		return e.RefreshContainers(full)
	}

	if len(containers) == 0 {
		// The container doesn't exist on the engine, remove it.
		e.Lock()
		delete(e.containers, ID)
		e.Unlock()

		return nil
	}

	_, err = e.updateContainer(containers[0], e.containers, full)
	return err
}

func (e *Engine) updateContainer(c dockerclient.Container, containers map[string]*Container, full bool) (map[string]*Container, error) {
	var container *Container

	e.RLock()
	if current, exists := e.containers[c.Id]; exists {
		// The container is already knowe.
		container = current
	} else {
		// This is a brand new container. We need to do a full refresh.
		container = &Container{
			Engine: e,
		}
		full = true
	}
	// Release the lock here as the next step is slow.
	// Trade-off: If updateContainer() is called concurrently for the same
	// container, we will end up doing a full refresh twice and the original
	// container (containers[container.Id]) will get replaced.
	e.RUnlock()

	// Update ContainerInfo.
	if full {
		info, err := e.client.InspectContainer(c.Id)
		if err != nil {
			return nil, err
		}
		// Convert the ContainerConfig from inspect into our own
		// cluster.ContainerConfig.
		container.Config = BuildContainerConfig(*info.Config)

		// FIXME remove "duplicate" lines and move this to cluster/config.go
		container.Config.CpuShares = container.Config.CpuShares * e.Cpus / 1024.0
		container.Config.HostConfig.CpuShares = container.Config.CpuShares

		// Save the entire inspect back into the container.
		container.Info = *info
	}

	// Update its internal state.
	e.Lock()
	container.Container = c
	containers[container.Id] = container
	e.Unlock()

	return containers, nil
}

func (e *Engine) refreshLoop() {
	for {
		var err error

		// Sleep stateRefreshPeriod or quit if we get stopped.
		select {
		case <-time.After(stateRefreshPeriod):
		case <-e.stopCh:
			return
		}

		err = e.RefreshContainers(false)
		if err == nil {
			err = e.RefreshImages()
		}

		if err != nil {
			if e.healthy {
				e.emitEvent("engine_disconnect")
			}
			e.healthy = false
			log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).Errorf("Flagging engine as dead. Updated state failed: %v", err)
		} else {
			if !e.healthy {
				log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).Info("Engine came back to life. Hooray!")
				e.client.StopAllMonitorEvents()
				e.client.StartMonitorEvents(e.handler, nil)
				e.emitEvent("engine_reconnect")
				if err := e.updateSpecs(); err != nil {
					log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).Errorf("Update engine specs failed: %v", err)
				}
			}
			e.healthy = true
		}
	}
}

func (e *Engine) emitEvent(event string) {
	// If there is no event handler registered, abort right now.
	if e.eventHandler == nil {
		return
	}
	ev := &Event{
		Event: dockerclient.Event{
			Status: event,
			From:   "swarm",
			Time:   time.Now().Unix(),
		},
		Engine: e,
	}
	e.eventHandler.Handle(ev)
}

// UsedMemory returns the sum of memory reserved by containers.
func (e *Engine) UsedMemory() int64 {
	var r int64
	e.RLock()
	for _, c := range e.containers {
		r += c.Config.Memory
	}
	e.RUnlock()
	return r
}

// UsedCpus returns the sum of CPUs reserved by containers.
func (e *Engine) UsedCpus() int64 {
	var r int64
	e.RLock()
	for _, c := range e.containers {
		r += c.Config.CpuShares
	}
	e.RUnlock()
	return r
}

// TotalMemory returns the total memory + overcommit
func (e *Engine) TotalMemory() int64 {
	return e.Memory + (e.Memory * e.overcommitRatio / 100)
}

// TotalCpus returns the total cpus + overcommit
func (e *Engine) TotalCpus() int64 {
	return e.Cpus + (e.Cpus * e.overcommitRatio / 100)
}

// Create a new container
func (e *Engine) Create(config *ContainerConfig, name string, pullImage bool) (*Container, error) {
	var (
		err    error
		id     string
		client = e.client
	)

	// Convert our internal ContainerConfig into something Docker will
	// understand.  Start by making a copy of the internal ContainerConfig as
	// we don't want to mess with the original.
	dockerConfig := config.ContainerConfig

	// nb of CPUs -> real CpuShares

	// FIXME remove "duplicate" lines and move this to cluster/config.go
	dockerConfig.CpuShares = int64(math.Ceil(float64(config.CpuShares*1024) / float64(e.Cpus)))
	dockerConfig.HostConfig.CpuShares = dockerConfig.CpuShares

	if id, err = client.CreateContainer(&dockerConfig, name); err != nil {
		// If the error is other than not found, abort immediately.
		if err != dockerclient.ErrNotFound || !pullImage {
			return nil, err
		}
		// Otherwise, try to pull the image...
		if err = e.Pull(config.Image, nil); err != nil {
			return nil, err
		}
		// ...And try agaie.
		if id, err = client.CreateContainer(&dockerConfig, name); err != nil {
			return nil, err
		}
	}

	// Register the container immediately while waiting for a state refresh.
	// Force a state refresh to pick up the newly created container.
	e.refreshContainer(id, true)

	e.RLock()
	defer e.RUnlock()

	return e.containers[id], nil
}

// RemoveContainer a container from the engine.
func (e *Engine) RemoveContainer(container *Container, force bool) error {
	if err := e.client.RemoveContainer(container.Id, force, true); err != nil {
		return err
	}

	// Remove the container from the state. Eventually, the state refresh loop
	// will rewrite this.
	e.Lock()
	defer e.Unlock()
	delete(e.containers, container.Id)

	return nil
}

// Pull an image on the engine
func (e *Engine) Pull(image string, authConfig *dockerclient.AuthConfig) error {
	if !strings.Contains(image, ":") {
		image = image + ":latest"
	}
	if err := e.client.PullImage(image, authConfig); err != nil {
		return err
	}

	// force refresh images
	e.RefreshImages()

	return nil
}

// Load an image on the engine
func (e *Engine) Load(reader io.Reader) error {
	if err := e.client.LoadImage(reader); err != nil {
		return err
	}

	// force fresh images
	e.RefreshImages()

	return nil
}

// Import image
func (e *Engine) Import(source string, repository string, tag string, imageReader io.Reader) error {
	if _, err := e.client.ImportImage(source, repository, tag, imageReader); err != nil {
		return err
	}

	// force fresh images
	e.RefreshImages()

	return nil
}

// RegisterEventHandler registers an event handler.
func (e *Engine) RegisterEventHandler(h EventHandler) error {
	if e.eventHandler != nil {
		return errors.New("event handler already set")
	}
	e.eventHandler = h
	return nil
}

// Containers returns all the containers in the engine.
func (e *Engine) Containers() Containers {
	e.RLock()
	containers := Containers{}
	for _, container := range e.containers {
		containers = append(containers, container)
	}
	e.RUnlock()
	return containers
}

// Images returns all the images in the engine
func (e *Engine) Images() []*Image {
	e.RLock()

	images := make([]*Image, 0, len(e.images))
	for _, image := range e.images {
		images = append(images, image)
	}
	e.RUnlock()
	return images
}

// Image returns the image with IDOrName in the engine
func (e *Engine) Image(IDOrName string) *Image {
	e.RLock()
	defer e.RUnlock()

	for _, image := range e.images {
		if image.Match(IDOrName, true) {
			return image
		}
	}
	return nil
}

func (e *Engine) String() string {
	return fmt.Sprintf("engine %s addr %s", e.ID, e.Addr)
}

func (e *Engine) handler(ev *dockerclient.Event, _ chan error, args ...interface{}) {
	// Something changed - refresh our internal state.
	switch ev.Status {
	case "pull", "untag", "delete":
		// These events refer to images so there's no need to update
		// containers.
		e.RefreshImages()
	case "die", "kill", "oom", "pause", "start", "stop", "unpause":
		// If the container state changes, we have to do an inspect in
		// order to update container.Info and get the new NetworkSettings.
		e.refreshContainer(ev.Id, true)
	default:
		// Otherwise, do a "soft" refresh of the container.
		e.refreshContainer(ev.Id, false)
	}

	// If there is no event handler registered, abort right now.
	if e.eventHandler == nil {
		return
	}

	event := &Event{
		Engine: e,
		Event:  *ev,
	}

	e.eventHandler.Handle(event)
}

// AddContainer inject a container into the internal state.
func (e *Engine) AddContainer(container *Container) error {
	e.Lock()
	defer e.Unlock()

	if _, ok := e.containers[container.Id]; ok {
		return errors.New("container already exists")
	}
	e.containers[container.Id] = container
	return nil
}

// Inject an image into the internal state.
func (e *Engine) addImage(image *Image) {
	e.Lock()
	defer e.Unlock()

	e.images = append(e.images, image)
}

// Remove a container from the internal test.
func (e *Engine) removeContainer(container *Container) error {
	e.Lock()
	defer e.Unlock()

	if _, ok := e.containers[container.Id]; !ok {
		return errors.New("container not found")
	}
	delete(e.containers, container.Id)
	return nil
}

// Wipes the internal container state.
func (e *Engine) cleanupContainers() {
	e.Lock()
	e.containers = make(map[string]*Container)
	e.Unlock()
}

// RenameContainer rename a container
func (e *Engine) RenameContainer(container *Container, newName string) error {
	// send rename request
	err := e.client.RenameContainer(container.Id, newName)
	if err != nil {
		return err
	}

	// refresh container
	return e.refreshContainer(container.Id, true)
}
