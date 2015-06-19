package mesos

import (
	"sync"
)

// Queue is a mesos task queue
type Queue struct {
	sync.Mutex

	c     *Cluster
	tasks map[string]*task
}

// Handler handles multiple tasks and returns the successfully handled tasks
type Handler interface {
	Process(tasks []*task) []*task
}

// NewQueue returns a new queue
func NewQueue(c *Cluster) *Queue {
	return &Queue{tasks: make(map[string]*task), c: c}
}

// Add tries to Do the task, if it's not possible, add the item to the queue for future tries
func (q *Queue) Add(task *task) {
	q.Lock()
	q.tasks[task.ID()] = task
	q.Unlock()
}

// Remove an item from the queue
func (q *Queue) Remove(tasks ...*task) {
	q.Lock()
	q.remove(tasks...)
	q.Unlock()
}

// Process tries to Do all the tasks in the queue and remove the tasks successfully done
func (q *Queue) Process(h Handler) {
	q.Lock()
	ts := []*task{}
	for _, t := range q.tasks {
		ts = append(ts, t)
	}
	toRemove := h.Process(ts)
	q.remove(toRemove...)
	q.Unlock()
}

func (q *Queue) remove(tasks ...*task) {
	for _, task := range tasks {
		delete(q.tasks, task.ID())
	}
}
