package cluster

import (
	"time"

	"github.com/docker/go-events"
	"github.com/docker/swarmkit/watch"
)

const (
	defaultEventQueueLimit   = 10000
	defaultEventQueueTimeout = 10 * time.Second
)

// APIEventsHandler broadcasts events to multiple client listeners.
type APIEventHandler struct {
	watchQueue *watch.Queue
}

// NewAPIEventHandler creates a new APIEventsHandler for a cluster.
// The new eventsHandler is initialized with no writers or channels.
func NewAPIEventHandler() *APIEventHandler {
	return &APIEventHandler{
		watchQueue: watch.NewQueue(watch.WithTimeout(defaultEventQueueTimeout), watch.WithLimit(defaultEventQueueLimit), watch.WithCloseOutChan()),
	}
}

// Add adds the writer and a new channel for the remote address.
func (eh *APIEventHandler) Watch() (eventq chan events.Event, cancel func()) {
	eventq, cancel = eh.watchQueue.Watch()
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
