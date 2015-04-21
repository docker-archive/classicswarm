package mesos

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/scheduler/node"
	"github.com/docker/swarm/scheduler/strategy"
	"github.com/docker/swarm/state"
	"github.com/mesos/mesos-go/mesosproto"
	mesosscheduler "github.com/mesos/mesos-go/scheduler"
	"github.com/samalba/dockerclient"
)

// Cluster struct for mesos
type Cluster struct {
	sync.RWMutex

	driver *mesosscheduler.MesosSchedulerDriver

	eventHandler cluster.EventHandler
	slaves       map[string]*slave
	scheduler    *scheduler.Scheduler
	options      *cluster.DriverOpts
	store        *state.Store
	TLSConfig    *tls.Config
	master       string
	pendingTasks *queue
	offerTimeout time.Duration
}

var (
	frameworkName    = "swarm"
	dockerDaemonPort = "2375"
	errNotSupported  = errors.New("not supported with mesos")
)

// NewCluster for mesos Cluster creation
func NewCluster(scheduler *scheduler.Scheduler, store *state.Store, TLSConfig *tls.Config, master string, options cluster.DriverOpts) (cluster.Cluster, error) {
	log.WithFields(log.Fields{"name": "mesos"}).Debug("Initializing cluster")

	cluster := &Cluster{
		slaves:       make(map[string]*slave),
		scheduler:    scheduler,
		options:      &options,
		store:        store,
		master:       master,
		TLSConfig:    TLSConfig,
		offerTimeout: 10 * time.Minute,
	}

	cluster.pendingTasks = &queue{c: cluster}

	// Empty string is accepted by the scheduler.
	user, _ := options.String("mesos.user", "SWARM_MESOS_USER")

	driverConfig := mesosscheduler.DriverConfig{
		Scheduler: cluster,
		Framework: &mesosproto.FrameworkInfo{Name: &frameworkName, User: &user},
		Master:    cluster.master,
	}

	// Changing port for https
	if cluster.TLSConfig != nil {
		dockerDaemonPort = "2376"
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
	task, err := newTask(config, name)

	if err != nil {
		return nil, err
	}

	go c.pendingTasks.add(task)

	select {
	case container := <-task.container:
		return container, nil
	case err := <-task.error:
		return nil, err
	case <-time.After(5 * time.Second):
		c.pendingTasks.remove(true, task.TaskId.GetValue())
		return nil, strategy.ErrNoResourcesAvailable
	}
}

func (c *Cluster) monitorTask(task *task) (bool, error) {
	taskStatus := task.getStatus()

	switch taskStatus.GetState() {
	case mesosproto.TaskState_TASK_STAGING:
	case mesosproto.TaskState_TASK_STARTING:
	case mesosproto.TaskState_TASK_RUNNING:
	case mesosproto.TaskState_TASK_FINISHED:
		return true, nil
	case mesosproto.TaskState_TASK_FAILED:
		return true, errors.New(taskStatus.GetMessage())
	case mesosproto.TaskState_TASK_KILLED:
		return true, nil
	case mesosproto.TaskState_TASK_LOST:
		return true, errors.New(taskStatus.GetMessage())
	case mesosproto.TaskState_TASK_ERROR:
		return true, errors.New(taskStatus.GetMessage())
	}

	return false, nil
}

// RemoveContainer to remove containers on mesos cluster
func (c *Cluster) RemoveContainer(container *cluster.Container, force bool) error {
	return nil
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

// Containers returns all the containers in the cluster.
func (c *Cluster) Containers() []*cluster.Container {
	c.RLock()
	defer c.RUnlock()

	out := []*cluster.Container{}
	for _, s := range c.slaves {
		for _, container := range s.engine.Containers() {
			if name := container.Config.NamespacedLabel("mesos.name"); name != "" {
				container.Names = append([]string{"/" + name}, container.Names...)
			}
			out = append(out, container)
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

	containers := c.Containers()

	// Match exact or short Container ID.
	for _, container := range containers {
		if container.Id == IDOrName || stringid.TruncateID(container.Id) == IDOrName {
			return container
		}
	}

	// Match exact Swarm ID.
	for _, container := range containers {
		if swarmID := container.Config.SwarmID(); swarmID == IDOrName || stringid.TruncateID(swarmID) == IDOrName {
			return container
		}
	}

	candidates := []*cluster.Container{}

	// Match name, /name or engine/name.
	for _, container := range containers {
		for _, name := range container.Names {
			if name == IDOrName || name == "/"+IDOrName || container.Engine.ID+name == IDOrName || container.Engine.Name+name == IDOrName {
				return container
			}
		}
	}

	if size := len(candidates); size == 1 {
		return candidates[0]
	} else if size > 1 {
		return nil
	}

	// Match Container ID prefix.
	for _, container := range containers {
		if strings.HasPrefix(container.Id, IDOrName) {
			candidates = append(candidates, container)
		}
	}

	// Match Swarm ID prefix.
	for _, container := range containers {
		if strings.HasPrefix(container.Config.SwarmID(), IDOrName) {
			candidates = append(candidates, container)
		}
	}

	if len(candidates) == 1 {
		if name := candidates[0].Config.NamespacedLabel("mesos.name"); name != "" {
			candidates[0].Names = append([]string{"/" + name}, candidates[0].Names...)
		}
		return candidates[0]
	}

	return nil
}

// RemoveImage removes an image from the cluster
func (c *Cluster) RemoveImage(image *cluster.Image) ([]*dockerclient.ImageDelete, error) {
	return nil, nil
}

// Pull will pull images on the cluster nodes
func (c *Cluster) Pull(name string, authConfig *dockerclient.AuthConfig, callback func(what, status string)) {

}

// Load images
func (c *Cluster) Load(imageReader io.Reader, callback func(what, status string)) {

}

// RenameContainer Rename a container
func (c *Cluster) RenameContainer(container *cluster.Container, newName string) error {
	return errNotSupported
}

func scalarResourceValue(offers map[string]*mesosproto.Offer, name string) float64 {
	var value float64
	for _, offer := range offers {
		for _, resource := range offer.Resources {
			if *resource.Name == name {
				value += *resource.Scalar.Value
			}
		}
	}
	return value
}

// listNodes returns all the nodess in the cluster.
func (c *Cluster) listNodes() []*node.Node {
	c.RLock()
	defer c.RUnlock()

	out := []*node.Node{}
	for _, s := range c.slaves {
		n := node.NewNode(s.engine)
		n.ID = s.id
		n.TotalCpus = int64(scalarResourceValue(s.offers, "cpus"))
		n.UsedCpus = 0
		n.TotalMemory = int64(scalarResourceValue(s.offers, "mem")) * 1024 * 1024
		n.UsedMemory = 0
		out = append(out, n)
	}
	return out
}

func (c *Cluster) listOffers() []*mesosproto.Offer {
	list := []*mesosproto.Offer{}
	for _, s := range c.slaves {
		for _, offer := range s.offers {
			list = append(list, offer)
		}
	}
	return list
}

// Info gives minimal information about containers and resources on the mesos cluster
func (c *Cluster) Info() [][2]string {
	offers := c.listOffers()
	info := [][2]string{
		{"\bStrategy", c.scheduler.Strategy()},
		{"\bFilters", c.scheduler.Filters()},
		{"\bOffers", fmt.Sprintf("%d", len(offers))},
	}

	sort.Sort(offerSorter(offers))

	for _, offer := range offers {
		info = append(info, [2]string{" Offer", offer.Id.GetValue()})
		for _, resource := range offer.Resources {
			info = append(info, [2]string{"  └ " + *resource.Name, fmt.Sprintf("%v", resource)})
		}
	}

	return info
}

// Registered method for registered mesos framework
func (c *Cluster) Registered(driver mesosscheduler.SchedulerDriver, fwID *mesosproto.FrameworkID, masterInfo *mesosproto.MasterInfo) {
	log.WithFields(log.Fields{"name": "mesos", "frameworkId": fwID.GetValue()}).Debug("Framework registered")
}

// Reregistered method for registered mesos framework
func (c *Cluster) Reregistered(mesosscheduler.SchedulerDriver, *mesosproto.MasterInfo) {
	log.WithFields(log.Fields{"name": "mesos"}).Debug("Framework re-registered")
}

// Disconnected method
func (c *Cluster) Disconnected(mesosscheduler.SchedulerDriver) {
	log.WithFields(log.Fields{"name": "mesos"}).Debug("Framework disconnected")
}

// ResourceOffers method
func (c *Cluster) ResourceOffers(_ mesosscheduler.SchedulerDriver, offers []*mesosproto.Offer) {
	log.WithFields(log.Fields{"name": "mesos", "offers": len(offers)}).Debug("Offers received")

	for _, offer := range offers {
		slaveID := offer.SlaveId.GetValue()
		s, ok := c.slaves[slaveID]
		if !ok {
			engine := cluster.NewEngine(*offer.Hostname+":"+dockerDaemonPort, 0)
			if err := engine.Connect(c.TLSConfig); err != nil {
				log.Error(err)
			} else {
				s = newSlave(slaveID, engine)
				c.slaves[slaveID] = s
			}
		}
		c.addOffer(offer)
	}
	c.pendingTasks.resourcesAdded()
}

func (c *Cluster) addOffer(offer *mesosproto.Offer) {
	s, ok := c.slaves[offer.SlaveId.GetValue()]
	if !ok {
		return
	}
	s.addOffer(offer)
	go func(offer *mesosproto.Offer) {
		<-time.After(c.offerTimeout)
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

// OfferRescinded method
func (c *Cluster) OfferRescinded(mesosscheduler.SchedulerDriver, *mesosproto.OfferID) {
}

// StatusUpdate method
func (c *Cluster) StatusUpdate(_ mesosscheduler.SchedulerDriver, taskStatus *mesosproto.TaskStatus) {
	log.WithFields(log.Fields{"name": "mesos", "state": taskStatus.State.String()}).Debug("Status update")
	taskID := taskStatus.TaskId.GetValue()
	slaveID := taskStatus.SlaveId.GetValue()
	s, ok := c.slaves[slaveID]
	if !ok {
		return
	}
	if task, ok := s.tasks[taskID]; ok {
		task.sendStatus(taskStatus)
	} else {
		var reason = ""
		if taskStatus.Reason != nil {
			reason = taskStatus.GetReason().String()
		}

		log.WithFields(log.Fields{
			"name":    "mesos",
			"state":   taskStatus.State.String(),
			"slaveId": taskStatus.SlaveId.GetValue(),
			"reason":  reason,
		}).Warn("Status update received for unknown slave")
	}
}

// FrameworkMessage method
func (c *Cluster) FrameworkMessage(mesosscheduler.SchedulerDriver, *mesosproto.ExecutorID, *mesosproto.SlaveID, string) {
}

// SlaveLost method
func (c *Cluster) SlaveLost(mesosscheduler.SchedulerDriver, *mesosproto.SlaveID) {
}

// ExecutorLost method
func (c *Cluster) ExecutorLost(mesosscheduler.SchedulerDriver, *mesosproto.ExecutorID, *mesosproto.SlaveID, int) {
}

// Error method
func (c *Cluster) Error(d mesosscheduler.SchedulerDriver, msg string) {
	log.WithFields(log.Fields{"name": "mesos"}).Error(msg)
}

// RANDOMENGINE returns a random engine.
func (c *Cluster) RANDOMENGINE() (*cluster.Engine, error) {
	n, err := c.scheduler.SelectNodeForContainer(c.listNodes(), &cluster.ContainerConfig{})
	if err != nil {
		return nil, err
	}
	if n != nil {
		return c.slaves[n.ID].engine, nil
	}
	return nil, nil
}
