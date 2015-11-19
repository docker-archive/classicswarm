package api

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

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
	timer.Stop()
	if until > 0 {
		dur := time.Unix(until, 0).Sub(time.Now())
		timer = time.NewTimer(dur)
	}

	select {
	case <-eh.cs[remoteAddr]:
	case <-timer.C: // `--until` timeout
		close(eh.cs[remoteAddr])
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
	eh.RLock()

	str := fmt.Sprintf("{%q:%q,%q:%q,%q:%q,%q:%d,%q:{%q:%q,%q:%q,%q:%q,%q:%q}}",
		"status", e.Status,
		"id", e.Id,
		"from", e.From+" node:"+e.Engine.Name,
		"time", e.Time,
		"node",
		"Name", e.Engine.Name,
		"Id", e.Engine.ID,
		"Addr", e.Engine.Addr,
		"Ip", e.Engine.IP)

	var failed []string

	for key, w := range eh.ws {
		if _, err := fmt.Fprintf(w, str); err != nil {
			// collect them to handle later under Lock
			failed = append(failed, key)
			continue
		}

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
	eh.RUnlock()
	eh.Lock()
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
