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

func (eh *eventsHandler) Handle(e *cluster.Event) error {
	eh.RLock()

	str := fmt.Sprintf("{%q:%q,%q:%q,%q:%q,%q:%d,%q:%q,%q:%q,%q:%q,%q:%q}",
		"status", e.Status,
		"id", e.Id,
		"from", e.From+" node:"+e.NodeName,
		"time", e.Time,
		"node_name", e.NodeName,
		"node_id", e.NodeID,
		"node_addr", e.NodeAddr,
		"node_ip", e.NodeIP)

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
