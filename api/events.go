package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/swarm/cluster"
)

// EventsHandler broadcasts events to multiple client listeners.
type eventsHandler struct {
	sync.RWMutex
	ws map[string]io.Writer
	cs map[string]chan struct{}
}

// NewEventsHandler creates a new EventsHandler for a cluster.
// The new eventsHandler is initialized with no writers or channels.
func newEventsHandler() *eventsHandler {
	return &eventsHandler{
		ws: make(map[string]io.Writer),
		cs: make(map[string]chan struct{}),
	}
}

// Add adds the writer and a new channel for the remote address.
func (eh *eventsHandler) Add(remoteAddr string, w io.Writer) {
	eh.Lock()
	eh.ws[remoteAddr] = w
	eh.cs[remoteAddr] = make(chan struct{})
	eh.Unlock()
}

// Wait waits on a signal from the remote address.
func (eh *eventsHandler) Wait(remoteAddr string, until int64) {

	timer := time.NewTimer(0)

	// Based on issue https://github.com/golang/go/issues/14383.
	// If timer has already expired, `time.Stop` will return false.
	// And we have to drain the channel manually.
	if !timer.Stop() {
		<-timer.C
	}

	if until > 0 {
		dur := time.Unix(until, 0).Sub(time.Now())
		timer = time.NewTimer(dur)
	}

	// subscribe to http client close event
	eh.RLock()
	w := eh.ws[remoteAddr]
	ch := eh.cs[remoteAddr]
	eh.RUnlock()
	var closeNotify <-chan bool
	if closeNotifier, ok := w.(http.CloseNotifier); ok {
		closeNotify = closeNotifier.CloseNotify()
	}

	select {
	case <-ch:
	case <-closeNotify:
	case <-timer.C: // `--until` timeout
		close(ch)
	}
	eh.cleanupHandler(remoteAddr)
}

func (eh *eventsHandler) cleanupHandler(remoteAddr string) {
	eh.Lock()
	// the maps are expected to have the same keys
	delete(eh.cs, remoteAddr)
	delete(eh.ws, remoteAddr)
	eh.Unlock()

}

// Handle writes information about a cluster event to each remote address in the cluster that has been added to the events handler.
// After an unsuccessful write to a remote address, the associated channel is closed and the address is removed from the events handler.
func (eh *eventsHandler) Handle(e *cluster.Event) error {
	ecopy := &cluster.Event{
		Message: events.Message{
			Status: e.Status,
			ID:     e.ID,
			From:   e.From,
			Type:   e.Type,
			Action: e.Action,
			Actor: events.Actor{
				ID:         e.Actor.ID,
				Attributes: make(map[string]string),
			},
			Time:     e.Time,
			TimeNano: e.TimeNano,
		},
	}

	// remove this hack once 1.10 is broadly adopted
	ecopy.From = e.From + " node:" + e.Engine.Name

	// Attributes will be nil if the event was sent by engine < 1.10
	if e.Actor.Attributes != nil {
		for k, v := range e.Actor.Attributes {
			ecopy.Actor.Attributes[k] = v
		}
	} else {
		ecopy.Actor.Attributes["node.name"] = e.Engine.Name
		ecopy.Actor.Attributes["node.id"] = e.Engine.ID
		ecopy.Actor.Attributes["node.addr"] = e.Engine.Addr
		ecopy.Actor.Attributes["node.ip"] = e.Engine.IP
	}

	data, err := json.Marshal(ecopy)
	if err != nil {
		return err
	}

	// remove the node field once 1.10 is broadly adopted & interlock stop relying on it
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

	var failed []string

	eh.Lock()

	for key, w := range eh.ws {
		if _, err := fmt.Fprint(w, string(data)); err != nil {
			// collect them to handle later under Lock
			failed = append(failed, key)
			continue
		}

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
	if len(failed) > 0 {
		for _, key := range failed {
			if ch, ok := eh.cs[key]; ok {
				close(ch)
			}
			delete(eh.cs, key)
		}
	}

	eh.Unlock()

	return nil
}

// Size returns the number of remote addresses that the events handler currently contains.
func (eh *eventsHandler) Size() int {
	eh.RLock()
	defer eh.RUnlock()
	return len(eh.ws)
}
