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
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/cluster/mesos/task"
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

	dockerEnginePort    string
	eventHandlers       *cluster.EventHandlers
	master              string
	agents              map[string]*agent
	scheduler           *Scheduler
	TLSConfig           *tls.Config
	options             *cluster.DriverOpts
	offerTimeout        time.Duration
	refuseTimeout       time.Duration
	taskCreationTimeout time.Duration
	pendingTasks        *task.Tasks
	engineOpts          *cluster.EngineOpts
}

const (
	frameworkName              = "swarm"
	defaultDockerEnginePort    = "2375"
	defaultDockerEngineTLSPort = "2376"
	dockerPortAttribute        = "docker_port"
	defaultOfferTimeout        = 30 * time.Second
	defaultRefuseTimeout       = 5 * time.Second
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
		eventHandlers:       cluster.NewEventHandlers(),
		master:              master,
		agents:              make(map[string]*agent),
		TLSConfig:           TLSConfig,
		options:             &options,
		offerTimeout:        defaultOfferTimeout,
		taskCreationTimeout: defaultTaskCreationTimeout,
		engineOpts:          engineOptions,
		refuseTimeout:       defaultRefuseTimeout,
	}

	cluster.pendingTasks = task.NewTasks(cluster)

	// Empty string is accepted by the scheduler.
	user, _ := options.String("mesos.user", "SWARM_MESOS_USER")

	// Override the hostname here because mesos-go will try
	// to shell out to the hostname binary and it won't work with our official image.
	// Do not check error here, so mesos-go can still try.
	hostname, _ := os.Hostname()

	driverConfig := mesosscheduler.DriverConfig{
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

	if checkpointFailover, ok := options.Bool("mesos.checkpointfailover", "SWARM_MESOS_CHECKPOINT_FAILOVER"); ok {
		driverConfig.Framework.Checkpoint = &checkpointFailover
	}

	if offerTimeout, ok := options.String("mesos.offertimeout", "SWARM_MESOS_OFFER_TIMEOUT"); ok {
		d, err := time.ParseDuration(offerTimeout)
		if err != nil {
			return nil, err
		}
		cluster.offerTimeout = d
	}

	if refuseTimeout, ok := options.String("mesos.offerrefusetimeout", "SWARM_MESOS_OFFER_REFUSE_TIMEOUT"); ok {
		d, err := time.ParseDuration(refuseTimeout)
		if err != nil {
			return nil, err
		}
		cluster.refuseTimeout = d
	}

	sched, err := NewScheduler(driverConfig, cluster, scheduler)
	if err != nil {
		return nil, err
	}

	cluster.scheduler = sched
	status, err := sched.driver.Start()
	if err != nil {
		log.Debugf("Mesos driver started, status/err %v: %v", status, err)
		return nil, err
	}
	log.Debugf("Mesos driver started, status %v", status)

	go func() {
		status, err := sched.driver.Join()
		log.Debugf("Mesos driver stopped unexpectedly, status/err %v: %v", status, err)

	}()

	return cluster, nil
}

// Handle callbacks for the events
func (c *Cluster) Handle(e *cluster.Event) error {
	c.eventHandlers.Handle(e)
	return nil
}

// RegisterEventHandler registers an event handler.
func (c *Cluster) RegisterEventHandler(h cluster.EventHandler) error {
	return c.eventHandlers.RegisterEventHandler(h)
}

// UnregisterEventHandler unregisters a previously registered event handler.
func (c *Cluster) UnregisterEventHandler(h cluster.EventHandler) {
	c.eventHandlers.UnregisterEventHandler(h)
}

// StartContainer starts a container
func (c *Cluster) StartContainer(container *cluster.Container, hostConfig *dockerclient.HostConfig) error {
	// if the container was started less than a second ago in detach mode, do not start it
	if time.Now().Unix()-container.Created > 1 || container.Config.Labels[cluster.SwarmLabelNamespace+".mesos.detach"] != "true" {
		return container.Engine.StartContainer(container.Id, hostConfig)
	}
	return nil
}

// CreateContainer for container creation in Mesos task
func (c *Cluster) CreateContainer(config *cluster.ContainerConfig, name string, authConfig *dockerclient.AuthConfig) (*cluster.Container, error) {
	if config.Memory == 0 && config.CpuShares == 0 {
		return nil, errResourcesNeeded
	}

	task, err := task.NewTask(config, name, c.taskCreationTimeout)
	if err != nil {
		return nil, err
	}

	go c.pendingTasks.Add(task)

	select {
	case container := <-task.GetContainer():
		return formatContainer(container), nil
	case err := <-task.Error:
		c.pendingTasks.Remove(task)
		return nil, err
	}
}

// RemoveContainer removes containers on mesos cluster
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
func (c *Cluster) RemoveImages(name string, force bool) ([]types.ImageDelete, error) {
	return nil, errNotSupported
}

// CreateNetwork creates a network in the cluster
func (c *Cluster) CreateNetwork(request *types.NetworkCreate) (*types.NetworkCreateResponse, error) {
	var (
		parts  = strings.SplitN(request.Name, "/", 2)
		config = &cluster.ContainerConfig{}
	)

	if len(parts) == 2 {
		// a node was specified, create the container only on this node
		request.Name = parts[1]
		config = cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:node==" + parts[0]}})
	}

	c.scheduler.Lock()
	nodes, err := c.scheduler.SelectNodesForContainer(c.listNodes(), config)
	c.scheduler.Unlock()
	if err != nil {
		return nil, err
	}
	if nodes == nil {
		return nil, errors.New("cannot find node to create network")
	}
	n := nodes[0]
	s, ok := c.agents[n.ID]
	if !ok {
		return nil, fmt.Errorf("Unable to create network on agent %q", n.ID)
	}
	resp, err := s.engine.CreateNetwork(request)
	c.refreshNetworks()
	return resp, err
}

func (c *Cluster) refreshNetworks() {
	var wg sync.WaitGroup
	for _, s := range c.agents {
		e := s.engine
		wg.Add(1)
		go func(e *cluster.Engine) {
			e.RefreshNetworks()
			wg.Done()
		}(e)
	}
	wg.Wait()
}

// CreateVolume creates a volume in the cluster
func (c *Cluster) CreateVolume(request *types.VolumeCreateRequest) (*cluster.Volume, error) {
	return nil, errNotSupported
}

// RemoveNetwork removes network from the cluster
func (c *Cluster) RemoveNetwork(network *cluster.Network) error {
	err := network.Engine.RemoveNetwork(network)
	c.refreshNetworks()
	return err
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
func (c *Cluster) RemoveImage(image *cluster.Image) ([]types.ImageDelete, error) {
	return nil, errNotSupported
}

// Pull pulls images on the cluster nodes
func (c *Cluster) Pull(name string, authConfig *dockerclient.AuthConfig, callback func(where, status string, err error)) {

}

// Load images
func (c *Cluster) Load(imageReader io.Reader, callback func(where, status string, err error)) {

}

// Import image
func (c *Cluster) Import(source string, repository string, tag string, imageReader io.Reader, callback func(what, status string, err error)) {

}

// RenameContainer renames a container
func (c *Cluster) RenameContainer(container *cluster.Container, newName string) error {
	//FIXME this doesn't work as the next refreshcontainer will erase this change (this change is in-memory only)
	container.Config.Labels[cluster.SwarmLabelNamespace+".mesos.name"] = newName

	return nil
}

// Networks returns all the networks in the cluster.
func (c *Cluster) Networks() cluster.Networks {
	c.RLock()
	defer c.RUnlock()

	out := cluster.Networks{}
	for _, s := range c.agents {
		out = append(out, s.engine.Networks()...)
	}

	return out

}

// Volumes returns all the volumes in the cluster.
func (c *Cluster) Volumes() cluster.Volumes {
	return nil
}

// listNodes returns all the nodes in the cluster.
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

// TotalMemory returns the total memory of the cluster
func (c *Cluster) TotalMemory() int64 {
	c.RLock()
	defer c.RUnlock()
	var totalMemory int64
	for _, s := range c.agents {
		totalMemory += int64(sumScalarResourceValue(s.offers, "mem")) * 1024 * 1024
	}
	return totalMemory
}

// TotalCpus returns the total memory of the cluster
func (c *Cluster) TotalCpus() int {
	c.RLock()
	defer c.RUnlock()
	var totalCpus int
	for _, s := range c.agents {
		totalCpus += int(sumScalarResourceValue(s.offers, "cpus"))
	}
	return totalCpus
}

// Info gives minimal information about containers and resources on the mesos cluster
func (c *Cluster) Info() [][2]string {
	offers := c.listOffers()
	info := [][2]string{
		{"Strategy", c.scheduler.Strategy()},
		{"Filters", c.scheduler.Filters()},
		{"Offers", fmt.Sprintf("%d", len(offers))},
	}

	sort.Sort(offerSorter(offers))

	for _, offer := range offers {
		info = append(info, [2]string{"  Offer", offer.Id.GetValue()})
		for _, resource := range offer.Resources {
			info = append(info, [2]string{"   └ " + resource.GetName(), formatResource(resource)})
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
			if _, err := c.scheduler.driver.DeclineOffer(offer.Id, &mesosproto.Filters{}); err != nil {
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
		s.engine.Disconnect()
		delete(c.agents, offer.SlaveId.GetValue())
	}
	return found
}

// LaunchTask selects node and calls driver to launch a task
func (c *Cluster) LaunchTask(t *task.Task) bool {
	c.scheduler.Lock()
	//change to explicit lock defer c.scheduler.Unlock()

	nodes, err := c.scheduler.SelectNodesForContainer(c.listNodes(), t.GetConfig())
	if err != nil {
		c.scheduler.Unlock()
		return false
	}
	n := nodes[0]
	s, ok := c.agents[n.ID]
	if !ok {
		t.Error <- fmt.Errorf("Unable to create on agent %q", n.ID)
		c.scheduler.Unlock()
		return true
	}

	// build the offer from its internal config and set the agentID

	c.Lock()
	// TODO: Only use the offer we need
	offerIDs := []*mesosproto.OfferID{}
	for _, offer := range c.agents[n.ID].offers {
		offerIDs = append(offerIDs, offer.Id)
	}

	t.Build(n.ID, c.agents[n.ID].offers)

	offerFilters := &mesosproto.Filters{}
	refuseSeconds := c.refuseTimeout.Seconds()
	offerFilters.RefuseSeconds = &refuseSeconds

	if _, err := c.scheduler.driver.LaunchTasks(offerIDs, []*mesosproto.TaskInfo{&t.TaskInfo}, offerFilters); err != nil {
		// TODO: Do not erase all the offers, only the one used
		for _, offer := range s.offers {
			c.removeOffer(offer)
		}
		c.Unlock()
		c.scheduler.Unlock()
		t.Error <- err
		return true
	}

	s.addTask(t)

	// TODO: Do not erase all the offers, only the one used
	for _, offer := range s.offers {
		c.removeOffer(offer)
	}
	c.Unlock()
	c.scheduler.Unlock()
	// block until we get the container
	finished, data, err := t.Monitor()
	taskID := t.TaskInfo.TaskId.GetValue()
	if err != nil {
		//remove task
		s.removeTask(taskID)
		t.Error <- err
		return true
	}
	if !finished {
		go func() {
			for {
				finished, _, err := t.Monitor()
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
			if !t.Stopped() {
				t.SetContainer(container)
			}
			return true
		}
	}

	log.Debug("Cannot parse docker info from task status, please upgrade Mesos to the latest version")
	// For mesos <= 0.22 we fallback to a full refresh + using labels
	// TODO: once 0.23 or 0.24 is released, remove all this block of code as it
	// doesn't scale very well.
	s.engine.RefreshContainers(true)

	for _, container := range s.engine.Containers() {
		if container.Config.Labels[cluster.SwarmLabelNamespace+".mesos.task"] == taskID {
			if !t.Stopped() {
				t.SetContainer(container)
			}
			return true
		}
	}

	if !t.Stopped() {
		t.Error <- fmt.Errorf("Container failed to create")
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

// BuildImage builds an image
func (c *Cluster) BuildImage(buildImage *types.ImageBuildOptions, out io.Writer) error {
	c.scheduler.Lock()

	// get an engine
	config := &cluster.ContainerConfig{dockerclient.ContainerConfig{
		CpuShares: buildImage.CPUShares,
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

// TagImage tags an image
func (c *Cluster) TagImage(IDOrName string, repo string, tag string, force bool) error {
	return errNotSupported
}
