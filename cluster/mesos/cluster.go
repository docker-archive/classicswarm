package mesos

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/scheduler/node"
	"github.com/docker/swarm/scheduler/strategy"
	"github.com/docker/swarm/state"
	"github.com/gogo/protobuf/proto"
	"github.com/mesos/mesos-go/mesosproto"
	mesosscheduler "github.com/mesos/mesos-go/scheduler"
	"github.com/samalba/dockerclient"
)

// Cluster struct for mesos
type Cluster struct {
	sync.RWMutex

	driver           *mesosscheduler.MesosSchedulerDriver
	dockerEnginePort string
	eventHandler     cluster.EventHandler
	master           string
	slaves           map[string]*slave
	scheduler        *scheduler.Scheduler
	store            *state.Store
	TLSConfig        *tls.Config
	options          *cluster.DriverOpts
	offerTimeout     time.Duration
	pendingTasks     *Queue
}

const (
	frameworkName              = "swarm"
	defaultDockerEnginePort    = "2375"
	defaultDockerEngineTLSPort = "2376"
	defaultOfferTimeout        = 10 * time.Minute
	taskCreationTimeout        = 30 * time.Second
)

var (
	errNotSupported = errors.New("not supported with mesos")
)

// NewCluster for mesos Cluster creation
func NewCluster(scheduler *scheduler.Scheduler, store *state.Store, TLSConfig *tls.Config, master string, options cluster.DriverOpts) (cluster.Cluster, error) {
	log.WithFields(log.Fields{"name": "mesos"}).Debug("Initializing cluster")

	cluster := &Cluster{
		dockerEnginePort: defaultDockerEnginePort,
		master:           master,
		slaves:           make(map[string]*slave),
		scheduler:        scheduler,
		store:            store,
		TLSConfig:        TLSConfig,
		options:          &options,
		offerTimeout:     defaultOfferTimeout,
	}

	cluster.pendingTasks = NewQueue(cluster)

	// Empty string is accepted by the scheduler.
	user, _ := options.String("mesos.user", "SWARM_MESOS_USER")

	driverConfig := mesosscheduler.DriverConfig{
		Scheduler: cluster,
		Framework: &mesosproto.FrameworkInfo{Name: proto.String(frameworkName), User: &user},
		Master:    cluster.master,
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
			return nil, fmt.Errorf("invalid address %s", bindingAddress)
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

// RegisterEventHandler registers an event handler.
func (c *Cluster) RegisterEventHandler(h cluster.EventHandler) error {
	if c.eventHandler != nil {
		return errors.New("event handler already set")
	}
	c.eventHandler = h
	return nil
}

// CreateContainer for container creation in Mesos task
func (c *Cluster) CreateContainer(config *cluster.ContainerConfig, name string) (*cluster.Container, error) {
	t, err := newTask(c, config, name)
	if err != nil {
		return nil, err
	}

	go func(t *task) {
		if len(c.Process([]*task{t})) == 0 {
			c.pendingTasks.Add(t)
		}
	}(t)

	select {
	case container := <-t.container:
		return formatContainer(container), nil
	case err := <-t.error:
		return nil, err
	case <-time.After(taskCreationTimeout):
		c.pendingTasks.Remove(t)
		return nil, strategy.ErrNoResourcesAvailable
	}
}

// RemoveContainer to remove containers on mesos cluster
func (c *Cluster) RemoveContainer(container *cluster.Container, force bool) error {
	return container.Engine.RemoveContainer(container, force)
}

// Images returns all the images in the cluster.
func (c *Cluster) Images() []*cluster.Image {
	c.RLock()
	defer c.RUnlock()

	out := []*cluster.Image{}
	for _, s := range c.slaves {
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

	for _, s := range c.slaves {
		if image := s.engine.Image(IDOrName); image != nil {
			return image
		}
	}
	return nil
}

// RemoveImages removes images from the cluster
func (c *Cluster) RemoveImages(name string) ([]*dockerclient.ImageDelete, error) {
	return nil, errNotSupported
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
	for _, s := range c.slaves {
		for _, container := range s.engine.Containers() {
			out = append(out, formatContainer(container))
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

	return formatContainer(cluster.Containers(c.Containers()).Get(IDOrName))
}

// RemoveImage removes an image from the cluster
func (c *Cluster) RemoveImage(image *cluster.Image) ([]*dockerclient.ImageDelete, error) {
	return nil, errNotSupported
}

// Pull will pull images on the cluster nodes
func (c *Cluster) Pull(name string, authConfig *dockerclient.AuthConfig, callback func(what, status string)) {

}

// Load images
func (c *Cluster) Load(imageReader io.Reader, callback func(what, status string)) {

}

// Import image
func (c *Cluster) Import(source string, repository string, tag string, imageReader io.Reader, callback func(what, status string)) {

}

// RenameContainer Rename a container
func (c *Cluster) RenameContainer(container *cluster.Container, newName string) error {
	//FIXME this doesn't work as the next refreshcontainer will erase this change (this change is in-memory only)
	container.Config.Labels[cluster.SwarmLabelNamespace+".mesos.name"] = newName

	return nil
}

// listNodes returns all the nodess in the cluster.
func (c *Cluster) listNodes() []*node.Node {
	c.RLock()
	defer c.RUnlock()

	out := []*node.Node{}
	for _, s := range c.slaves {
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
	for _, s := range c.slaves {
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
	for _, s := range c.slaves {
		totalMemory += int64(sumScalarResourceValue(s.offers, "mem")) * 1024 * 1024
	}
	return totalMemory
}

// TotalCpus return the total memory of the cluster
func (c *Cluster) TotalCpus() int64 {
	c.RLock()
	defer c.RUnlock()
	var totalCpus int64
	for _, s := range c.slaves {
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
	c.Lock()
	defer c.Unlock()

	s, ok := c.slaves[offer.SlaveId.GetValue()]
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

func (c *Cluster) _removeOffer(offer *mesosproto.Offer) bool {
	log.WithFields(log.Fields{"name": "mesos", "offerID": offer.Id.String()}).Debug("Removing offer")
	s, ok := c.slaves[offer.SlaveId.GetValue()]
	if !ok {
		return false
	}
	found := s.removeOffer(offer.Id.GetValue())
	if s.empty() {
		// Disconnect from engine
		delete(c.slaves, offer.SlaveId.GetValue())
	}
	return found
}

func (c *Cluster) removeOffer(offer *mesosproto.Offer) bool {
	c.Lock()
	defer c.Unlock()

	return c._removeOffer(offer)
}

func (c *Cluster) placeTask(task *task, nodes []*node.Node) *slave {
	n, err := c.scheduler.SelectNodeForContainer(nodes, task.config)
	if err != nil {
		return nil
	}

	s, ok := c.slaves[n.ID]
	if !ok {
		task.error <- fmt.Errorf("Unable to create on slave %q", n.ID)
		return nil
	}

	task.build(n.ID)
	n.TotalCpus -= task.config.CpuShares
	n.TotalMemory -= task.config.Memory
	s.addTask(task)
	return s
}

func (c *Cluster) Process(tasks []*task) []*task {
	return c.scheduleTasks(tasks)
}

// scheduleTasks schedules and launches the tasks to mesos driver, and return the tasks
// that was successfully placed.
func (c *Cluster) scheduleTasks(tasks []*task) []*task {
	// Keep a map of slaves to tasks scheduled
	usedSlaves := make(map[*slave][]*task)
	scheduled := []*task{}
	taskInfos := []*mesosproto.TaskInfo{}
	nodes := c.listNodes()
	for _, t := range tasks {
		if s := c.placeTask(t, nodes); s != nil {
			ts, ok := usedSlaves[s]
			if !ok {
				ts = []*task{t}
				usedSlaves[s] = ts
			} else {
				ts = append(ts, t)
			}
			scheduled = append(scheduled, t)
			taskInfos = append(taskInfos, &t.TaskInfo)
		}
	}

	c.Lock()
	// TODO: Only use the offer we need from a slave
	offerIDs := []*mesosproto.OfferID{}
	for s := range usedSlaves {
		for _, offer := range c.slaves[s.id].offers {
			offerIDs = append(offerIDs, offer.Id)
			c._removeOffer(offer)
		}
	}

	if _, err := c.driver.LaunchTasks(offerIDs, taskInfos, &mesosproto.Filters{}); err != nil {
		for s, ts := range usedSlaves {
			for _, t := range ts {
				taskID := t.TaskInfo.TaskId.GetValue()
				s.removeTask(taskID)
				t.error <- err
			}
		}
		c.Unlock()
		return scheduled
	}

	c.Unlock()

	for s, ts := range usedSlaves {
		for _, t := range ts {
			go func(s *slave, t *task) {
				taskID := t.TaskInfo.TaskId.GetValue()
				_, data, err := t.monitor()
				// Once we received the first update we no longer need to track the task.
				s.removeTask(taskID)
				if err != nil {
					// remove task
					t.error <- err
					return
				}

				// Register the container immediately while waiting for a state refresh.

				// In mesos 0.23+ the docker inspect will be sent back in the taskStatus.Data
				// We can use this to find the right container.
				inspect := []dockerclient.ContainerInfo{}
				if data != nil && json.Unmarshal(data, &inspect) != nil && len(inspect) == 1 {
					container := &cluster.Container{Container: dockerclient.Container{Id: inspect[0].Id}, Engine: s.engine}
					if container.Refresh() == nil {
						t.container <- container
						return
					}
				}

				log.Debug("Cannot parse docker info from task status, please upgrade Mesos to the last version")
				// For mesos <= 0.22 we fallback to a full refresh + using labels
				// TODO: once 0.23 or 0.24 is released, remove all this block of code as it
				// doesn't scale very well.
				s.engine.RefreshContainers(true)

				for _, container := range s.engine.Containers() {
					if container.Config.Labels[cluster.SwarmLabelNamespace+".mesos.task"] == taskID {
						t.container <- container
						return
					}
				}

				t.error <- fmt.Errorf("Container failed to create")
			}(s, t)
		}
	}
	return scheduled
}

// RANDOMENGINE returns a random engine.
func (c *Cluster) RANDOMENGINE() (*cluster.Engine, error) {
	c.RLock()
	defer c.RUnlock()

	n, err := c.scheduler.SelectNodeForContainer(c.listNodes(), &cluster.ContainerConfig{})
	if err != nil {
		return nil, err
	}
	if n != nil {
		return c.slaves[n.ID].engine, nil
	}
	return nil, nil
}
