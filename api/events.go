package api

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/docker/libcluster/swarm"
)

type eventsHandler struct {
	sync.RWMutex
	ws map[string]io.Writer
	cs map[string]chan struct{}
}

func (eh *eventsHandler) Handle(e *swarm.Event) error {
	eh.RLock()
	str := fmt.Sprintf("{%q:%q,%q:%q,%q:%q,%q:%d}", "status", e.Type, "id", e.Container.Id, "from", e.Container.Image+" node:"+e.Node.ID, "time", e.Time.Unix())

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
