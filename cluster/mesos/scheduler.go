package mesos

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler"
	"github.com/mesos/mesos-go/mesosproto"
	mesosscheduler "github.com/mesos/mesos-go/scheduler"
)

// Scheduler structure for mesos driver
type Scheduler struct {
	scheduler.Scheduler

	driver  *mesosscheduler.MesosSchedulerDriver
	cluster *Cluster
}

// NewScheduler for Scheduler mesos driver creation
func NewScheduler(config mesosscheduler.DriverConfig, cluster *Cluster, sched *scheduler.Scheduler) (*Scheduler, error) {
	scheduler := Scheduler{
		Scheduler: *sched,
		cluster:   cluster,
	}

	config.Scheduler = &scheduler
	driver, err := mesosscheduler.NewMesosSchedulerDriver(config)
	if err != nil {
		return nil, err
	}
	scheduler.driver = driver
	return &scheduler, nil
}

// Registered method for registered mesos framework
func (s *Scheduler) Registered(driver mesosscheduler.SchedulerDriver, fwID *mesosproto.FrameworkID, masterInfo *mesosproto.MasterInfo) {
	log.WithFields(log.Fields{"name": "mesos", "frameworkId": fwID.GetValue()}).Debug("Framework registered")
}

// Reregistered method for registered mesos framework
func (s *Scheduler) Reregistered(mesosscheduler.SchedulerDriver, *mesosproto.MasterInfo) {
	log.WithFields(log.Fields{"name": "mesos"}).Debug("Framework re-registered")
}

// Disconnected method
func (s *Scheduler) Disconnected(mesosscheduler.SchedulerDriver) {
	log.WithFields(log.Fields{"name": "mesos"}).Debug("Framework disconnected")
}

// ResourceOffers method
func (s *Scheduler) ResourceOffers(_ mesosscheduler.SchedulerDriver, offers []*mesosproto.Offer) {
	log.WithFields(log.Fields{"name": "mesos", "offers": len(offers)}).Debug("Offers received")

	for _, offer := range offers {
		agentID := offer.SlaveId.GetValue()
		dockerPort := s.cluster.dockerEnginePort

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

		a, ok := s.cluster.agents[agentID]
		if !ok {
			engine := cluster.NewEngine(*offer.Hostname+":"+dockerPort, 0, s.cluster.engineOpts)
			if err := engine.Connect(s.cluster.TLSConfig); err != nil {
				log.Error(err)
			} else {
				// Set engine state to healthy and start refresh loop
				engine.ValidationComplete()
				a = newAgent(agentID, engine)
				s.cluster.agents[agentID] = a
				if err := a.engine.RegisterEventHandler(s.cluster); err != nil {
					log.Error(err)
				}
			}

		}
		s.cluster.addOffer(offer)

	}
	go s.cluster.pendingTasks.Process()
}

// OfferRescinded method
func (s *Scheduler) OfferRescinded(_ mesosscheduler.SchedulerDriver, offerID *mesosproto.OfferID) {
	log.WithFields(log.Fields{"name": "mesos", "OfferID": offerID.GetValue()}).Debug("Offer Rescinded")

	for _, agent := range s.cluster.agents {
		if offer, ok := agent.offers[offerID.GetValue()]; ok {
			s.cluster.removeOffer(offer)
			break
		}
	}
}

// StatusUpdate method
func (s *Scheduler) StatusUpdate(_ mesosscheduler.SchedulerDriver, taskStatus *mesosproto.TaskStatus) {
	log.WithFields(log.Fields{"name": "mesos", "state": taskStatus.State.String()}).Debug("Status update")
	taskID := taskStatus.TaskId.GetValue()
	agentID := taskStatus.SlaveId.GetValue()
	a, ok := s.cluster.agents[agentID]
	if !ok {
		return
	}
	if task, ok := a.tasks[taskID]; ok {
		task.SendStatus(taskStatus)
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
func (s *Scheduler) FrameworkMessage(mesosscheduler.SchedulerDriver, *mesosproto.ExecutorID, *mesosproto.SlaveID, string) {
}

// SlaveLost method
func (s *Scheduler) SlaveLost(mesosscheduler.SchedulerDriver, *mesosproto.SlaveID) {
}

// ExecutorLost method
func (s *Scheduler) ExecutorLost(mesosscheduler.SchedulerDriver, *mesosproto.ExecutorID, *mesosproto.SlaveID, int) {
}

// Error method
func (s *Scheduler) Error(d mesosscheduler.SchedulerDriver, msg string) {
	log.WithFields(log.Fields{"name": "mesos"}).Error(msg)
}
