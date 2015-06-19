package mesos

import (
	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/mesos/mesos-go/mesosproto"
	mesosscheduler "github.com/mesos/mesos-go/scheduler"
)

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
			engine := cluster.NewEngine(*offer.Hostname+":"+c.dockerEnginePort, 0)
			if err := engine.Connect(c.TLSConfig); err != nil {
				log.Error(err)
			} else {
				s = newSlave(slaveID, engine)
				c.slaves[slaveID] = s
			}
		}
		c.addOffer(offer)
	}
	go c.pendingTasks.Process(c)
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
