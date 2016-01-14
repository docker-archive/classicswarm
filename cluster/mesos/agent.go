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
	s.offers[offer.Id.GetValue()] = offer
	s.Unlock()
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
