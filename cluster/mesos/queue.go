package mesos

import (
	"fmt"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/mesos/mesos-go/mesosproto"
)

type queue struct {
	sync.RWMutex
	tasks []*task
	c     *Cluster
}

func (q *queue) scheduleTask(t *task) bool {
	n, err := q.c.scheduler.SelectNodeForContainer(q.c.listNodes(), t.config)
	if err != nil {
		return false
	}
	s, ok := q.c.slaves[n.ID]
	if !ok {
		t.error <- fmt.Errorf("Unable to create on slave %q", n.ID)
		return true
	}

	// build the offer from it's internal config and set the slaveID
	t.build(n.ID)

	q.c.Lock()
	// TODO: Only use the offer we need
	offerIds := []*mesosproto.OfferID{}
	for _, offer := range q.c.slaves[n.ID].offers {
		offerIds = append(offerIds, offer.Id)
	}

	if _, err := q.c.driver.LaunchTasks(offerIds, []*mesosproto.TaskInfo{&t.TaskInfo}, &mesosproto.Filters{}); err != nil {
		// TODO: Do not erase all the offers, only the one used
		for _, offer := range s.offers {
			q.c.removeOffer(offer)
		}
		s.Unlock()
		t.error <- err
		return true
	}

	s.addTask(t)

	// TODO: Do not erase all the offers, only the one used
	for _, offer := range s.offers {
		q.c.removeOffer(offer)
	}
	q.c.Unlock()
	// block until we get the container
	finished, err := q.c.monitorTask(t)

	if err != nil {
		//remove task
		s.removeTask(t.TaskInfo.TaskId.GetValue())
		t.error <- err
		return true
	}
	if !finished {
		go func() {
			for {
				finished, err := q.c.monitorTask(t)
				if err != nil {
					// TODO proper error message
					log.Error(err)
					break
				}
				if finished {
					break
				}
			}
			//remove task
		}()
	}

	// Register the container immediately while waiting for a state refresh.
	// Force a state refresh to pick up the newly created container.
	// FIXME: unexport this method, see FIXME in engine.go
	s.engine.RefreshContainers(true)

	// TODO: We have to return the right container that was just created.
	// Once we receive the ContainerID from the executor.
	for _, container := range s.engine.Containers() {
		t.container <- container
		// TODO save in store
		return true
	}

	t.error <- fmt.Errorf("Container failed to create")
	return true
}

func (q *queue) add(t *task) {
	q.Lock()
	defer q.Unlock()

	if !q.scheduleTask(t) {
		q.tasks = append(q.tasks, t)
	}
}

func (q *queue) remove(lock bool, taskIDs ...string) {
	if lock {
		q.Lock()
		defer q.Unlock()
	}

	new := []*task{}
	for _, t := range q.tasks {
		found := false
		for _, taskID := range taskIDs {
			if t.TaskId.GetValue() == taskID {
				found = true
			}
		}
		if !found {
			new = append(new, t)
		}
	}
	q.tasks = new
}

func (q *queue) resourcesAdded() {
	go q.process()
}

func (q *queue) process() {
	q.Lock()
	defer q.Unlock()

	ids := []string{}
	for _, t := range q.tasks {
		if q.scheduleTask(t) {
			ids = append(ids, t.TaskId.GetValue())
		}
	}

	q.remove(false, ids...)
}
