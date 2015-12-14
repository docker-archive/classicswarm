package mesos

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/cluster/mesos/queue"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/scheduler/node"
	"github.com/gogo/protobuf/proto"
	"github.com/mesos/mesos-go/mesosproto"
	mesosscheduler "github.com/mesos/mesos-go/scheduler"
	"github.com/samalba/dockerclient"
)

// Cluster struct for mesos
type Cluster struct {
	sync.RWMutex

	driver              *mesosscheduler.MesosSchedulerDriver
	dockerEnginePort    string
	eventHandler        cluster.EventHandler
	master              string
	agents              map[string]*agent
	scheduler           *scheduler.Scheduler
	TLSConfig           *tls.Config
	options             *cluster.DriverOpts
	offerTimeout        time.Duration
	taskCreationTimeout time.Duration
	pendingTasks        *queue.Queue
	engineOpts          *cluster.EngineOpts
}

const (
	frameworkName              = "swarm"
	defaultDockerEnginePort    = "2375"
	defaultDockerEngineTLSPort = "2376"
	dockerPortAttribute        = "docker_port"
	defaultOfferTimeout        = 30 * time.Second
	defaultTaskCreationTimeout = 5 * time.Second
)

var (
	errNotSupported    = errors.New("not supported with mesos")
	errResourcesNeeded = errors.New("resources constraints (-c and/or -m) are required by mesos")
)

// NewCluster for mesos Cluster creation
func NewCluster(scheduler *scheduler.Scheduler, TLSConfig *tls.Config, master string, options cluster.DriverOpts, engineOptions *cluster.EngineOpts) (cluster.Cluster, error) {
	log.WithFields(log.Fields{"name": "mesos"}).Debug("Initializing cluster")

	// Enabling mesos-go glog logging
	if log.GetLevel() == log.DebugLevel {
		flag.Lookup("logtostderr").Value.Set("true")
	}
	cluster := &Cluster{
		dockerEnginePort:    defaultDockerEnginePort,
		master:              master,
		agents:              make(map[string]*agent),
		scheduler:           scheduler,
		TLSConfig:           TLSConfig,
		options:             &options,
		offerTimeout:        defaultOfferTimeout,
		taskCreationTimeout: defaultTaskCreationTimeout,
		engineOpts:          engineOptions,
	}

	cluster.pendingTasks = queue.NewQueue()

	// Empty string is accepted by the scheduler.
	user, _ := options.String("mesos.user", "SWARM_MESOS_USER")

	// Override the hostname here because mesos-go will try
	// to shell out to the hostname binary and it won't work with our official image.
	// Do not check error here, so mesos-go can still try.
	hostname, _ := os.Hostname()

	driverConfig := mesosscheduler.DriverConfig{
		Scheduler:        cluster,
		Framework:        &mesosproto.FrameworkInfo{Name: proto.String(frameworkName), User: &user},
		Master:           cluster.master,
		HostnameOverride: hostname,
	}

	if taskCreationTimeout, ok := options.String("mesos.tasktimeout", "SWARM_MESOS_TASK_TIMEOUT"); ok {
		d, err := time.ParseDuration(taskCreationTimeout)
		if err != nil {
			return nil, err
		}
		cluster.taskCreationTimeout = d
	}
	// Changing port for https
	if cluster.TLSConfig != nil {
		cluster.dockerEnginePort = defaultDockerEngineTLSPort
	}

	if bindingPort, ok := options.Uint("mesos.port", "SWARM_MESOS_PORT"); ok {
		driverConfig.BindingPort = uint16(bindingPort)
	}

	if bindingAddress, ok := options.IP("mesos.address", "SWARM_MESOS_ADDRESS"); ok {
		if bindingAddress == nil {
			value, _ := options.String("mesos.address", "SWARM_MESOS_ADDRESS")
			return nil, fmt.Errorf(
				"invalid IP address for cluster-opt mesos.address: \"%s\"",
				value)
		}
		driverConfig.BindingAddress = bindingAddress
	}

	if offerTimeout, ok := options.String("mesos.offertimeout", "SWARM_MESOS_OFFER_TIMEOUT"); ok {
		d, err := time.ParseDuration(offerTimeout)
		if err != nil {
			return nil, err
		}
		cluster.offerTimeout = d
	}

	driver, err := mesosscheduler.NewMesosSchedulerDriver(driverConfig)
	if err != nil {
		return nil, err
	}

	cluster.driver = driver

	status, err := driver.Start()
	if err != nil {
		log.Debugf("Mesos driver started, status/err %v: %v", status, err)
		return nil, err
	}
	log.Debugf("Mesos driver started, status %v", status)

	return cluster, nil
}

// Handle callbacks for the events
func (c *Cluster) Handle(e *cluster.Event) error {
	if c.eventHandler == nil {
		return nil
	}
	if err := c.eventHandler.Handle(e); err != nil {
		log.Error(err)
	}
	return nil
}

// RegisterEventHandler registers an event handler.
func (c *Cluster) RegisterEventHandler(h cluster.EventHandler) error {
	if c.eventHandler != nil {
		return errors.New("event handler already set")
	}
	c.eventHandler = h
	return nil
}

// CreateContainer for container creation in Mesos task
func (c *Cluster) CreateContainer(config *cluster.ContainerConfig, name string, authConfig *dockerclient.AuthConfig) (*cluster.Container, error) {
	if config.Memory == 0 && config.CpuShares == 0 {
		return nil, errResourcesNeeded
	}

	task, err := newTask(c, config, name)
	if err != nil {
		return nil, err
	}

	go c.pendingTasks.Add(task)

	select {
	case container := <-task.container:
		return formatContainer(container), nil
	case err := <-task.error:
		return nil, err
	case <-time.After(c.taskCreationTimeout):
		c.pendingTasks.Remove(task)
		return nil, fmt.Errorf("container failed to start after %s", c.taskCreationTimeout)
	}
}

// RemoveContainer to remove containers on mesos cluster
func (c *Cluster) RemoveContainer(container *cluster.Container, force, volumes bool) error {
	c.scheduler.Lock()
	defer c.scheduler.Unlock()

	return container.Engine.RemoveContainer(container, force, volumes)
}

// Images returns all the images in the cluster.
func (c *Cluster) Images() cluster.Images {
	c.RLock()
	defer c.RUnlock()

	out := []*cluster.Image{}
	for _, s := range c.agents {
		out = append(out, s.engine.Images()...)
	}
	return out
}

// Image returns an image with IdOrName in the cluster
func (c *Cluster) Image(IDOrName string) *cluster.Image {
	// Abort immediately if the name is empty.
	if len(IDOrName) == 0 {
		return nil
	}

	c.RLock()
	defer c.RUnlock()

	for _, s := range c.agents {
		if image := s.engine.Image(IDOrName); image != nil {
			return image
		}
	}
	return nil
}

// RemoveImages removes images from the cluster
func (c *Cluster) RemoveImages(name string, force bool) ([]*dockerclient.ImageDelete, error) {
	return nil, errNotSupported
}

// CreateNetwork creates a network in the cluster
func (c *Cluster) CreateNetwork(request *dockerclient.NetworkCreate) (*dockerclient.NetworkCreateResponse, error) {
	return nil, errNotSupported
}

// CreateVolume creates a volume in the cluster
func (c *Cluster) CreateVolume(request *dockerclient.VolumeCreateRequest) (*cluster.Volume, error) {
	return nil, errNotSupported
}

// RemoveNetwork removes network from the cluster
func (c *Cluster) RemoveNetwork(network *cluster.Network) error {
	return errNotSupported
}

// RemoveVolumes removes volumes from the cluster
func (c *Cluster) RemoveVolumes(name string) (bool, error) {
	return false, errNotSupported
}

func formatContainer(container *cluster.Container) *cluster.Container {
	if container == nil {
		return nil
	}
	if name := container.Config.Labels[cluster.SwarmLabelNamespace+".mesos.name"]; name != "" && container.Names[0] != "/"+name {
		container.Names = append([]string{"/" + name}, container.Names...)
	}
	return container
}

// Containers returns all the containers in the cluster.
func (c *Cluster) Containers() cluster.Containers {
	c.RLock()
	defer c.RUnlock()

	out := cluster.Containers{}
	for _, s := range c.agents {
		for _, container := range s.engine.Containers() {
			if container.Config.Labels != nil {
				if _, ok := container.Config.Labels[cluster.SwarmLabelNamespace+".mesos.task"]; ok {
					out = append(out, formatContainer(container))
				}
			}
		}
	}

	return out
}

// Container returns the container with IdOrName in the cluster
func (c *Cluster) Container(IDOrName string) *cluster.Container {
	// Abort immediately if the name is empty.
	if len(IDOrName) == 0 {
		return nil
	}

	c.RLock()
	defer c.RUnlock()

	return formatContainer(cluster.Containers(c.Containers()).Get(IDOrName))
}

// RemoveImage removes an image from the cluster
func (c *Cluster) RemoveImage(image *cluster.Image) ([]*dockerclient.ImageDelete, error) {
	return nil, errNotSupported
}

// Pull will pull images on the cluster nodes
func (c *Cluster) Pull(name string, authConfig *dockerclient.AuthConfig, callback func(where, status string, err error)) {

}

// Load images
func (c *Cluster) Load(imageReader io.Reader, callback func(where, status string, err error)) {

}

// Import image
func (c *Cluster) Import(source string, repository string, tag string, imageReader io.Reader, callback func(what, status string, err error)) {

}

// RenameContainer Rename a container
func (c *Cluster) RenameContainer(container *cluster.Container, newName string) error {
	//FIXME this doesn't work as the next refreshcontainer will erase this change (this change is in-memory only)
	container.Config.Labels[cluster.SwarmLabelNamespace+".mesos.name"] = newName

	return nil
}

// Networks returns all the networks in the cluster.
func (c *Cluster) Networks() cluster.Networks {
	return cluster.Networks{}
}

// Volumes returns all the volumes in the cluster.
func (c *Cluster) Volumes() []*cluster.Volume {
	return nil
}

// Volume returns the volume name in the cluster
func (c *Cluster) Volume(name string) *cluster.Volume {
	return nil
}

// listNodes returns all the nodess in the cluster.
func (c *Cluster) listNodes() []*node.Node {
	c.RLock()
	defer c.RUnlock()

	out := []*node.Node{}
	for _, s := range c.agents {
		n := node.NewNode(s.engine)
		n.ID = s.id
		n.TotalCpus = int64(sumScalarResourceValue(s.offers, "cpus"))
		n.UsedCpus = 0
		n.TotalMemory = int64(sumScalarResourceValue(s.offers, "mem")) * 1024 * 1024
		n.UsedMemory = 0
		out = append(out, n)
	}
	return out
}

func (c *Cluster) listOffers() []*mesosproto.Offer {
	c.RLock()
	defer c.RUnlock()

	list := []*mesosproto.Offer{}
	for _, s := range c.agents {
		for _, offer := range s.offers {
			list = append(list, offer)
		}
	}
	return list
}

// TotalMemory return the total memory of the cluster
func (c *Cluster) TotalMemory() int64 {
	c.RLock()
	defer c.RUnlock()
	var totalMemory int64
	for _, s := range c.agents {
		totalMemory += int64(sumScalarResourceValue(s.offers, "mem")) * 1024 * 1024
	}
	return totalMemory
}

// TotalCpus return the total memory of the cluster
func (c *Cluster) TotalCpus() int64 {
	c.RLock()
	defer c.RUnlock()
	var totalCpus int64
	for _, s := range c.agents {
		totalCpus += int64(sumScalarResourceValue(s.offers, "cpus"))
	}
	return totalCpus
}

// Info gives minimal information about containers and resources on the mesos cluster
func (c *Cluster) Info() [][]string {
	offers := c.listOffers()
	info := [][]string{
		{"\bStrategy", c.scheduler.Strategy()},
		{"\bFilters", c.scheduler.Filters()},
		{"\bOffers", fmt.Sprintf("%d", len(offers))},
	}

	sort.Sort(offerSorter(offers))

	for _, offer := range offers {
		info = append(info, []string{" Offer", offer.Id.GetValue()})
		for _, resource := range offer.Resources {
			info = append(info, []string{"  └ " + resource.GetName(), formatResource(resource)})
		}
	}

	return info
}

func (c *Cluster) addOffer(offer *mesosproto.Offer) {
	s, ok := c.agents[offer.SlaveId.GetValue()]
	if !ok {
		return
	}
	s.addOffer(offer)
	go func(offer *mesosproto.Offer) {
		time.Sleep(c.offerTimeout)
		// declining Mesos offers to make them available to other Mesos services
		if c.removeOffer(offer) {
			if _, err := c.driver.DeclineOffer(offer.Id, &mesosproto.Filters{}); err != nil {
				log.WithFields(log.Fields{"name": "mesos"}).Errorf("Error while declining offer %q: %v", offer.Id.GetValue(), err)
			} else {
				log.WithFields(log.Fields{"name": "mesos"}).Debugf("Offer %q declined successfully", offer.Id.GetValue())
			}
		}
	}(offer)
}

func (c *Cluster) removeOffer(offer *mesosproto.Offer) bool {
	log.WithFields(log.Fields{"name": "mesos", "offerID": offer.Id.String()}).Debug("Removing offer")
	s, ok := c.agents[offer.SlaveId.GetValue()]
	if !ok {
		return false
	}
	found := s.removeOffer(offer.Id.GetValue())
	if s.empty() {
		// Disconnect from engine
		delete(c.agents, offer.SlaveId.GetValue())
	}
	return found
}

func (c *Cluster) scheduleTask(t *task) bool {
	c.scheduler.Lock()
	defer c.scheduler.Unlock()

	nodes, err := c.scheduler.SelectNodesForContainer(c.listNodes(), t.config)
	if err != nil {
		return false
	}
	n := nodes[0]
	s, ok := c.agents[n.ID]
	if !ok {
		t.error <- fmt.Errorf("Unable to create on agent %q", n.ID)
		return true
	}

	// build the offer from it's internal config and set the agentID

	c.Lock()
	// TODO: Only use the offer we need
	offerIDs := []*mesosproto.OfferID{}
	for _, offer := range c.agents[n.ID].offers {
		offerIDs = append(offerIDs, offer.Id)
	}

	t.build(n.ID, c.agents[n.ID].offers)

	if _, err := c.driver.LaunchTasks(offerIDs, []*mesosproto.TaskInfo{&t.TaskInfo}, &mesosproto.Filters{}); err != nil {
		// TODO: Do not erase all the offers, only the one used
		for _, offer := range s.offers {
			c.removeOffer(offer)
		}
		c.Unlock()
		t.error <- err
		return true
	}

	s.addTask(t)

	// TODO: Do not erase all the offers, only the one used
	for _, offer := range s.offers {
		c.removeOffer(offer)
	}
	c.Unlock()
	// block until we get the container
	finished, data, err := t.monitor()
	taskID := t.TaskInfo.TaskId.GetValue()
	if err != nil {
		//remove task
		s.removeTask(taskID)
		t.error <- err
		return true
	}
	if !finished {
		go func() {
			for {
				finished, _, err := t.monitor()
				if err != nil {
					// TODO do a better log by sending proper error message
					log.Error(err)
					break
				}
				if finished {
					break
				}
			}
			//remove the task once it's finished
		}()
	}

	// Register the container immediately while waiting for a state refresh.

	// In mesos 0.23+ the docker inspect will be sent back in the taskStatus.Data
	// We can use this to find the right container.
	inspect := []dockerclient.ContainerInfo{}
	if data != nil && json.Unmarshal(data, &inspect) == nil && len(inspect) == 1 {
		container := &cluster.Container{Container: dockerclient.Container{Id: inspect[0].Id}, Engine: s.engine}
		if container, err := container.Refresh(); err == nil {
			if !t.done {
				t.container <- container
			}
			return true
		}
	}

	log.Debug("Cannot parse docker info from task status, please upgrade Mesos to the last version")
	// For mesos <= 0.22 we fallback to a full refresh + using labels
	// TODO: once 0.23 or 0.24 is released, remove all this block of code as it
	// doesn't scale very well.
	s.engine.RefreshContainers(true)

	for _, container := range s.engine.Containers() {
		if container.Config.Labels[cluster.SwarmLabelNamespace+".mesos.task"] == taskID {
			if !t.done {
				t.container <- container
			}
			return true
		}
	}

	if !t.done {
		t.error <- fmt.Errorf("Container failed to create")
	}
	return true
}

// RANDOMENGINE returns a random engine.
func (c *Cluster) RANDOMENGINE() (*cluster.Engine, error) {
	c.RLock()
	defer c.RUnlock()

	nodes, err := c.scheduler.SelectNodesForContainer(c.listNodes(), &cluster.ContainerConfig{})
	if err != nil {
		return nil, err
	}
	n := nodes[0]
	return c.agents[n.ID].engine, nil
}

// BuildImage build an image
func (c *Cluster) BuildImage(buildImage *dockerclient.BuildImage, out io.Writer) error {
	c.scheduler.Lock()

	// get an engine
	config := &cluster.ContainerConfig{dockerclient.ContainerConfig{
		CpuShares: buildImage.CpuShares,
		Memory:    buildImage.Memory,
	}}
	nodes, err := c.scheduler.SelectNodesForContainer(c.listNodes(), config)
	c.scheduler.Unlock()
	if err != nil {
		return err
	}
	n := nodes[0]

	reader, err := c.agents[n.ID].engine.BuildImage(buildImage)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, reader); err != nil {
		return err
	}

	c.agents[n.ID].engine.RefreshImages()
	return nil
}

// TagImage tag an image
func (c *Cluster) TagImage(IDOrName string, repo string, tag string, force bool) error {
	return errNotSupported
}
