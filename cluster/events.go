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

// EventsHandler broadcasts events to multiple client listeners.
type EventsHandler struct {
	watchQueue *watch.Queue
}

// NewEventsHandler creates a new EventsHandler for a cluster.
// The new eventsHandler is initialized with no writers or channels.
func NewEventsHandler() *EventsHandler {
	return &EventsHandler{
		watchQueue: watch.NewQueue(watch.WithTimeout(defaultEventQueueTimeout), watch.WithLimit(defaultEventQueueLimit), watch.WithCloseOutChan()),
	}
}

// Add adds the writer and a new channel for the remote address.
func (eh *EventsHandler) Watch() (eventq chan events.Event, cancel func()) {
	eventq, cancel = eh.watchQueue.Watch()
	return eventq, cancel
}

func (eh *EventsHandler) cleanupHandler(remoteAddr string) {
	eh.watchQueue.Close()
}

// Handle writes information about a cluster event to each remote address in the cluster that has been added to the events handler.
// After an unsuccessful write to a remote address, the associated channel is closed and the address is removed from the events handler.
func (eh *EventsHandler) Handle(e *Event) error {
	eh.watchQueue.Publish(e)
	return nil
}

// Size returns the number of remote addresses that the events handler currently contains.
func (eh *EventsHandler) Size() int {
	return 0
}
