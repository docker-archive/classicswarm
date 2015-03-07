package api

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/docker/swarm/cluster"
)

type eventsHandler struct {
	sync.RWMutex
	ws map[string]io.Writer
	cs map[string]chan struct{}
}

func NewEventsHandler() *eventsHandler {
	return &eventsHandler{
		ws: make(map[string]io.Writer),
		cs: make(map[string]chan struct{}),
	}
}

func (eh *eventsHandler) Add(remoteAddr string, w io.Writer) {
	eh.Lock()
	eh.ws[remoteAddr] = w
	eh.cs[remoteAddr] = make(chan struct{})
	eh.Unlock()
}

func (eh *eventsHandler) Wait(remoteAddr string) {
	<-eh.cs[remoteAddr]
}

func (eh *eventsHandler) Handle(e *cluster.Event) error {
	eh.RLock()

	str := fmt.Sprintf("{%q:%q,%q:%q,%q:%q,%q:%d,%q:%s}",
		"status", e.Status,
		"id", e.Id,
		"from", e.From+" node:"+e.Node.Name(),
		"time", e.Time,
		"node", cluster.SerializeNode(e.Node))

	for key, w := range eh.ws {
		if _, err := fmt.Fprintf(w, str); err != nil {
			close(eh.cs[key])
			delete(eh.ws, key)
			delete(eh.cs, key)
			continue
		}

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

	}
	eh.RUnlock()
	return nil
}

func (eh *eventsHandler) Size() int {
	eh.RLock()
	defer eh.RUnlock()
	return len(eh.ws)
}
