package queue

import "sync"

// Item represents a simple item in the queue
type Item interface {
	ID() string
	Do() bool
}

// Queue is a simple item queue
type Queue struct {
	sync.Mutex
	items map[string]Item
}

// NewQueue returns a new queue
func NewQueue() *Queue {
	return &Queue{items: make(map[string]Item)}
}

// Add tries to Do the item, if it's not possible, add the item to the queue for future tries
func (q *Queue) Add(item Item) {
	q.Lock()
	defer q.Unlock()

	if !item.Do() {
		q.items[item.ID()] = item
	}
}

// Remove an item from the queue
func (q *Queue) Remove(items ...Item) {
	q.Lock()
	defer q.Unlock()

	q.remove(items...)
}

// Process tries to Do all the items in the queue and remove the items successfully done
func (q *Queue) Process() {
	q.Lock()
	defer q.Unlock()

	toRemove := []Item{}
	for _, item := range q.items {
		if item.Do() {
			toRemove = append(toRemove, item)
		}
	}

	q.remove(toRemove...)
}

func (q *Queue) remove(items ...Item) {
	for _, item := range items {
		delete(q.items, item.ID())
	}
}
