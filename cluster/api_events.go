package cluster

import (
	"sync/atomic"
	"time"

	"github.com/docker/go-events"
	"github.com/docker/swarmkit/watch"
)

const (
	defaultEventQueueLimit   = 10000
	defaultEventQueueTimeout = 10 * time.Second
)

// APIEventHandler broadcasts events to multiple client listeners.
type APIEventHandler struct {
	listenerCount *uint64
	watchQueue    *watch.Queue
}

// NewAPIEventHandler creates a new APIEventsHandler for a cluster.
// The new eventsHandler is initialized with no writers or channels.
func NewAPIEventHandler() *APIEventHandler {
	count := uint64(0)
	return &APIEventHandler{
		listenerCount: &count,
		watchQueue:    watch.NewQueue(watch.WithTimeout(defaultEventQueueTimeout), watch.WithLimit(defaultEventQueueLimit), watch.WithCloseOutChan()),
	}
}

// Watch adds the writer and a new channel for the remote address.
func (eh *APIEventHandler) Watch() (chan events.Event, func()) {
	// create a new queue and subscribe to it
	eventq, cancelFunc := eh.watchQueue.Watch()
	// increment counter
	atomic.AddUint64(eh.listenerCount, 1)

	cancel := func() {
		// decrement counter
		atomic.AddUint64(eh.listenerCount, ^uint64(0))
		cancelFunc()
	}
	return eventq, cancel
}

func (eh *APIEventHandler) cleanupHandler() {
	eh.watchQueue.Close()
}

// Handle writes information about a cluster event to each remote address in the cluster that has been added to the events handler.
// After an unsuccessful write to a remote address, the associated channel is closed and the address is removed from the events handler.
func (eh *APIEventHandler) Handle(e *Event) error {
	eh.watchQueue.Publish(e)
	return nil
}

// Size returns the number of event queues currently listening for events
func (eh *APIEventHandler) Size() int {
	return int(atomic.LoadUint64(eh.listenerCount))
}
