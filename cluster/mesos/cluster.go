package mesos

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/units"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/scheduler/node"
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
	options      *cluster.Options
	store        *state.Store
}

var (
	frameworkName    = "swarm"
	dockerDaemonPort = "2375"
)

// NewCluster for mesos Cluster creation
func NewCluster(scheduler *scheduler.Scheduler, store *state.Store, options *cluster.Options) cluster.Cluster {
	log.WithFields(log.Fields{"name": "mesos"}).Debug("Initializing cluster")

	cluster := &Cluster{
		slaves:    make(map[string]*slave),
		scheduler: scheduler,
		options:   options,
		store:     store,
	}

	// Empty string is accepted by the scheduler.
	user := os.Getenv("SWARM_MESOS_USER")

	driverConfig := mesosscheduler.DriverConfig{
		Scheduler: cluster,
		Framework: &mesosproto.FrameworkInfo{Name: &frameworkName, User: &user},
		Master:    options.Discovery,
	}

	// Changing port for https
	if options.TLSConfig != nil {
		dockerDaemonPort = "2376"
	}

	bindingAddressEnv := os.Getenv("SWARM_MESOS_ADDRESS")
	bindingPortEnv := os.Getenv("SWARM_MESOS_PORT")

	if bindingPortEnv != "" {
		log.Debugf("SWARM_MESOS_PORT found, Binding port to %s", bindingPortEnv)
		bindingPort, err := strconv.ParseUint(bindingPortEnv, 0, 16)
		if err != nil {
			log.Errorf("Unable to parse SWARM_MESOS_PORT, error: %s", err)
			return nil
		}
		driverConfig.BindingPort = uint16(bindingPort)
	}

	if bindingAddressEnv != "" {
		log.Debugf("SWARM_MESOS_ADDRESS found, Binding address to %s", bindingAddressEnv)
		bindingAddress := net.ParseIP(bindingAddressEnv)
		if bindingAddress == nil {
			log.Error("Unable to parse SWARM_MESOS_ADDRESS")
			return nil
		}
		driverConfig.BindingAddress = bindingAddress
	}

	driver, err := mesosscheduler.NewMesosSchedulerDriver(driverConfig)
	if err != nil {
		return nil
	}

	cluster.driver = driver

	status, err := driver.Start()
	log.Debugf("Mesos driver started, status/err %v: %v", status, err)
	if err != nil {
		return nil
	}

	return cluster
}

// RegisterEventHandler registers an event handler.
func (c *Cluster) RegisterEventHandler(h cluster.EventHandler) error {
	if c.eventHandler != nil {
		return errors.New("event handler already set")
	}
	c.eventHandler = h
	return nil
}

// CreateContainer for container creation
func (c *Cluster) CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error) {

	n, err := c.scheduler.SelectNodeForContainer(c.listNodes(), config)
	if err != nil {
		return nil, err
	}

	if nn, ok := c.slaves[n.ID]; ok {
		container, err := nn.create(c.driver, config, name, true)
		if err != nil {
			return nil, err
		}

		if container == nil {
			return nil, fmt.Errorf("Container failed to create")
		}

		// TODO: do not store the container as it might be a wrong ContainerID
		// see TODO in slave.go
		//st := &state.RequestedState{
		//ID:     container.Id,
		//Name:   name,
		//Config: config,
		//}
		return container, nil //c.store.Add(container.Id, st)
	}
	return nil, nil
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
	for _, n := range c.slaves {
		out = append(out, n.Images()...)
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
	for _, n := range c.slaves {
		if image := n.Image(IDOrName); image != nil {
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
	for _, n := range c.slaves {
		out = append(out, n.Containers()...)
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
	for _, n := range c.slaves {
		if container := n.Container(IDOrName); container != nil {
			return container
		}
	}

	return nil
}

// RemoveImage removes an image from the cluster
func (c *Cluster) RemoveImage(image *cluster.Image) ([]*dockerclient.ImageDelete, error) {
	return nil, nil
}

// Pull will pull images on the cluster nodes
func (c *Cluster) Pull(name string, callback func(what, status string)) {

}

// listNodes returns all the nodess in the cluster.
func (c *Cluster) listNodes() []*node.Node {
	c.RLock()
	defer c.RUnlock()

	out := []*node.Node{}
	for _, s := range c.slaves {
		out = append(out, s.toNode())
	}

	return out
}

// listSlaves returns all the slaves in the cluster.
func (c *Cluster) listSlaves() []*slave {
	c.RLock()
	defer c.RUnlock()

	out := []*slave{}
	for _, s := range c.slaves {
		out = append(out, s)
	}
	return out
}

// Info gives minimal information about containers and resources on the mesos cluster
func (c *Cluster) Info() [][2]string {
	info := [][2]string{
		{"\bStrategy", c.scheduler.Strategy()},
		{"\bFilters", c.scheduler.Filters()},
		{"\bSlaves", fmt.Sprintf("%d", len(c.slaves))},
	}

	slaves := c.listSlaves()
	sort.Sort(SlaveSorter(slaves))

	for _, slave := range slaves {
		info = append(info, [2]string{slave.Name, slave.Addr})
		info = append(info, [2]string{" └ Containers", fmt.Sprintf("%d", len(slave.Containers()))})
		info = append(info, [2]string{" └ Reserved CPUs", fmt.Sprintf("%d / %d", slave.UsedCpus(), slave.TotalCpus())})
		info = append(info, [2]string{" └ Reserved Memory", fmt.Sprintf("%s / %s", units.BytesSize(float64(slave.UsedMemory())), units.BytesSize(float64(slave.TotalMemory())))})
		info = append(info, [2]string{" └ Offers", fmt.Sprintf("%d", len(slave.offers))})
		for _, offer := range slave.offers {
			info = append(info, [2]string{" Offer", offer.Id.GetValue()})
			for _, resource := range offer.Resources {
				info = append(info, [2]string{"  └ " + *resource.Name, fmt.Sprintf("%v", resource)})
			}
		}
	}

	return info
}

// Registered method for registered mesos framework
func (c *Cluster) Registered(driver mesosscheduler.SchedulerDriver, fwID *mesosproto.FrameworkID, masterInfo *mesosproto.MasterInfo) {
	log.Debugf("Swarm is registered with Mesos with framework id: %s", fwID.GetValue())
}

// Reregistered method for registered mesos framework
func (c *Cluster) Reregistered(mesosscheduler.SchedulerDriver, *mesosproto.MasterInfo) {
	log.Debug("Swarm is re-registered with Mesos")
}

// Disconnected method
func (c *Cluster) Disconnected(mesosscheduler.SchedulerDriver) {
	log.Debug("Swarm is disconnectd with Mesos")
}

// ResourceOffers method
func (c *Cluster) ResourceOffers(_ mesosscheduler.SchedulerDriver, offers []*mesosproto.Offer) {
	log.WithFields(log.Fields{"name": "mesos", "offers": len(offers)}).Debug("Offers received")

	for _, offer := range offers {
		slaveID := offer.SlaveId.GetValue()
		if slave, ok := c.slaves[slaveID]; ok {
			slave.addOffer(offer)
		} else {
			slave := newSlave(*offer.Hostname+":"+dockerDaemonPort, c.options.OvercommitRatio, offer)
			err := slave.Connect(c.options.TLSConfig)
			if err != nil {
				log.Error(err)
			} else {
				c.slaves[slaveID] = slave
			}
		}
	}
}

// OfferRescinded method
func (c *Cluster) OfferRescinded(mesosscheduler.SchedulerDriver, *mesosproto.OfferID) {
}

// StatusUpdate method
func (c *Cluster) StatusUpdate(_ mesosscheduler.SchedulerDriver, taskStatus *mesosproto.TaskStatus) {
	log.WithFields(log.Fields{"name": "mesos", "state": taskStatus.State.String()}).Debug("Status update")

	if slave, ok := c.slaves[taskStatus.SlaveId.GetValue()]; ok {
		if ch, ok := slave.statuses[taskStatus.TaskId.GetValue()]; ok {
			ch <- taskStatus
		}
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
	log.Error(msg)
}

// RANDOMENGINE returns a random engine.
func (c *Cluster) RANDOMENGINE() (*cluster.Engine, error) {
	n, err := c.scheduler.SelectNodeForContainer(c.listNodes(), &dockerclient.ContainerConfig{})
	if err != nil {
		return nil, err
	}
	if n != nil {
		return &c.slaves[n.ID].Engine, nil
	}
	return nil, nil
}
