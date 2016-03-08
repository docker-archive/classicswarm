package mesos

import (
	"sync"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/cluster/mesos/task"
	"github.com/mesos/mesos-go/mesosproto"
)

type agent struct {
	sync.RWMutex

	id     string
	offers map[string]*mesosproto.Offer
	tasks  map[string]*task.Task
	engine *cluster.Engine
}

func newAgent(sid string, e *cluster.Engine) *agent {
	return &agent{
		id:     sid,
		offers: make(map[string]*mesosproto.Offer),
		tasks:  make(map[string]*task.Task),
		engine: e,
	}
}

func (s *agent) addOffer(offer *mesosproto.Offer) {
	s.Lock()
	defer s.Unlock()
	s.offers[offer.Id.GetValue()] = offer

	// Updates the resource type of the corresponding Docker
	// engine with this new offer.
	s.updateEngineResourceType(offer)
}

func (s *agent) addTask(task *task.Task) {
	s.Lock()
	s.tasks[task.TaskInfo.TaskId.GetValue()] = task
	s.Unlock()
}

func (s *agent) removeOffer(offerID string) bool {
	s.Lock()
	defer s.Unlock()
	found := false
	_, found = s.offers[offerID]
	if found {
		delete(s.offers, offerID)

		// Afer an offer is removed from this agent, it needs to go throught
		// all remaining offers on this agent to reset the resource type of
		// the corresponding Docker engine.
		s.engine.Labels[engineResourceType] = UnknownResource
		for _, offer := range s.offers {
			s.updateEngineResourceType(offer)
		}
	}
	return found
}

func (s *agent) removeTask(taskID string) bool {
	s.Lock()
	defer s.Unlock()
	found := false
	_, found = s.tasks[taskID]
	if found {
		delete(s.tasks, taskID)
	}
	return found
}

func (s *agent) empty() bool {
	s.RLock()
	defer s.RUnlock()
	return len(s.offers) == 0 && len(s.tasks) == 0
}

func (s *agent) getOffers() map[string]*mesosproto.Offer {
	s.RLock()
	defer s.RUnlock()
	return s.offers
}

func (s *agent) getTasks() map[string]*task.Task {
	s.RLock()
	defer s.RUnlock()
	return s.tasks
}

// Updates the corresponding Docker engine resource type with this specified offer.
func (s *agent) updateEngineResourceType(offer *mesosproto.Offer) {
	if _, existed := s.engine.Labels[engineResourceType]; !existed {
		s.engine.Labels[engineResourceType] = UnknownResource
	}

	for _, resource := range offer.Resources {
		currentType := s.engine.Labels[engineResourceType]

		if resource.GetRevocable() == nil {
			switch currentType {
			case RevocableResourceOnly:
				s.engine.Labels[engineResourceType] = MixedResource
			case UnknownResource:
				s.engine.Labels[engineResourceType] = RegularResourceOnly
			}
		} else {
			switch currentType {
			case RegularResourceOnly:
				s.engine.Labels[engineResourceType] = MixedResource
			case UnknownResource:
				s.engine.Labels[engineResourceType] = RevocableResourceOnly
			}
		}
	}
}
