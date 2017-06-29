package api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/docker/go-events"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarmkit/watch"
)

const (
	defaultEventQueueLimit   = 10000
	defaultEventQueueTimeout = 10 * time.Second
)

// EventsHandler broadcasts events to multiple client listeners.
type eventsHandler struct {
	watchQueue *watch.Queue
}

// NewEventsHandler creates a new EventsHandler for a cluster.
// The new eventsHandler is initialized with no writers or channels.
func newEventsHandler() *eventsHandler {
	return &eventsHandler{
		watchQueue: watch.NewQueue(watch.WithTimeout(defaultEventQueueTimeout), watch.WithLimit(defaultEventQueueLimit), watch.WithCloseOutChan()),
	}
}

// Add adds the writer and a new channel for the remote address.
func (eh *eventsHandler) Watch() (eventq chan events.Event, cancel func()) {
	eventq, cancel = eh.watchQueue.Watch()
	return eventq, cancel
}

func (eh *eventsHandler) cleanupHandler(remoteAddr string) {
	eh.watchQueue.Close()
}

// Handle writes information about a cluster event to each remote address in the cluster that has been added to the events handler.
// After an unsuccessful write to a remote address, the associated channel is closed and the address is removed from the events handler.
func (eh *eventsHandler) Handle(e *cluster.Event) error {
	eh.watchQueue.Publish(e)
	return nil
}

// Size returns the number of remote addresses that the events handler currently contains.
func (eh *eventsHandler) Size() int {
	return 0
}

// normalizeEvent takes a cluster Event and ensures backward compatibility
// and all the right fields filled up
func normalizeEvent(receivedEvent *cluster.Event) ([]byte, error) {
	// make a local copy of the event
	e := *receivedEvent
	// make a fresh copy of the Actor.Attributes map to prevent a race condition
	e.Actor.Attributes = make(map[string]string)
	for k, v := range receivedEvent.Actor.Attributes {
		e.Actor.Attributes[k] = v
	}

	// remove this hack once 1.10 is broadly adopted
	e.From = e.From + " node:" + e.Engine.Name

	e.Actor.Attributes["node.name"] = e.Engine.Name
	e.Actor.Attributes["node.id"] = e.Engine.ID
	e.Actor.Attributes["node.addr"] = e.Engine.Addr
	e.Actor.Attributes["node.ip"] = e.Engine.IP

	data, err := json.Marshal(&e)
	if err != nil {
		return nil, err
	}

	// remove the node field once 1.10 is broadly adopted & interlock stops relying on it
	node := fmt.Sprintf(",%q:{%q:%q,%q:%q,%q:%q,%q:%q}}",
		"node",
		"Name", e.Engine.Name,
		"Id", e.Engine.ID,
		"Addr", e.Engine.Addr,
		"Ip", e.Engine.IP,
	)

	// insert Node field
	data = data[:len(data)-1]
	data = append(data, []byte(node)...)

	return data, nil
}
