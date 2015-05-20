package mesos

import (
	"sync"

	"github.com/docker/swarm/cluster"
	"github.com/mesos/mesos-go/mesosproto"
)

type slave struct {
	sync.RWMutex

	id     string
	offers map[string]*mesosproto.Offer
	tasks  map[string]*task
	engine *cluster.Engine
}

func newSlave(sid string, e *cluster.Engine) *slave {
	return &slave{
		id:     sid,
		offers: make(map[string]*mesosproto.Offer),
		tasks:  make(map[string]*task),
		engine: e,
	}
}

func (s *slave) addOffer(offer *mesosproto.Offer) {
	s.Lock()
	s.offers[offer.Id.GetValue()] = offer
	s.Unlock()
}

func (s *slave) addTask(task *task) {
	s.Lock()
	s.tasks[task.TaskInfo.TaskId.GetValue()] = task
	s.Unlock()
}

func (s *slave) removeOffer(offerID string) bool {
	s.Lock()
	defer s.Unlock()
	found := false
	_, found = s.offers[offerID]
	if found {
		delete(s.offers, offerID)
	}
	return found
}

func (s *slave) removeTask(taskID string) bool {
	s.Lock()
	defer s.Unlock()
	found := false
	_, found = s.tasks[taskID]
	if found {
		delete(s.tasks, taskID)
	}
	return found
}

func (s *slave) empty() bool {
	s.RLock()
	defer s.RUnlock()
	return len(s.offers) == 0 && len(s.tasks) == 0
}

func (s *slave) getOffers() map[string]*mesosproto.Offer {
	s.RLock()
	defer s.RUnlock()
	return s.offers
}

func (s *slave) getTasks() map[string]*task {
	s.RLock()
	defer s.RUnlock()
	return s.tasks
}
