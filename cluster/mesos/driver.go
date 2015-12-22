package mesos

import (
	"fmt"

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
		agentID := offer.SlaveId.GetValue()
		dockerPort := c.dockerEnginePort
		for _, attribute := range offer.GetAttributes() {
			if attribute.GetName() == dockerPortAttribute {
				switch attribute.GetType() {
				case mesosproto.Value_SCALAR:
					dockerPort = fmt.Sprintf("%d", int(attribute.GetScalar().GetValue()))
				case mesosproto.Value_TEXT:
					dockerPort = attribute.GetText().GetValue()
				}
			}
		}
		s, ok := c.agents[agentID]
		if !ok {
			engine := cluster.NewEngine(*offer.Hostname+":"+dockerPort, 0, c.engineOpts)
			if err := engine.Connect(c.TLSConfig); err != nil {
				log.Error(err)
			} else {
				// Set engine state to healthy and start refresh loop
				engine.ValidationComplete()
				s = newAgent(agentID, engine)
				c.agents[agentID] = s
				if err := s.engine.RegisterEventHandler(c); err != nil {
					log.Error(err)
				}
			}
		}
		c.addOffer(offer)
	}
	go c.pendingTasks.Process()
}

// OfferRescinded method
func (c *Cluster) OfferRescinded(mesosscheduler.SchedulerDriver, *mesosproto.OfferID) {
}

// StatusUpdate method
func (c *Cluster) StatusUpdate(_ mesosscheduler.SchedulerDriver, taskStatus *mesosproto.TaskStatus) {
	log.WithFields(log.Fields{"name": "mesos", "state": taskStatus.State.String()}).Debug("Status update")
	taskID := taskStatus.TaskId.GetValue()
	agentID := taskStatus.SlaveId.GetValue()
	s, ok := c.agents[agentID]
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
			"agentId": taskStatus.SlaveId.GetValue(),
			"reason":  reason,
		}).Warn("Status update received for unknown agent")
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
