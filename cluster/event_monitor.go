package cluster

import (
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/swarm/swarmclient"
	"golang.org/x/net/context"
)

//EventsMonitor monitors events
type EventsMonitor struct {
	stopChan    chan struct{}
	cli         swarmclient.SwarmAPIClient
	handler     func(msg events.Message) error
	lastEventAt time.Time
}

// NewEventsMonitor returns an EventsMonitor
func NewEventsMonitor(cli swarmclient.SwarmAPIClient, handler func(msg events.Message) error) *EventsMonitor {
	return &EventsMonitor{
		cli:         cli,
		handler:     handler,
		lastEventAt: time.Now().UTC(),
	}
}

// Start starts the EventsMonitor
func (em *EventsMonitor) Start(ec chan error) {
	em.stopChan = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	options := types.EventsOptions{
		Since: em.lastEventAt.Format(time.RFC3339),
	}
	responseStream, errStream := em.cli.Events(ctx, options)

	go func() {
		defer cancel()
		for {
			select {
			case event := <-responseStream:
				if err := em.handler(event); err != nil {
					ec <- err
					return
				}
				// if event stream is broken, it should use `--since` in restart to pick up from last event
				em.lastEventAt = time.Unix(0, event.TimeNano+1).UTC()
			case err := <-errStream:
				ec <- err
				return
			case <-em.stopChan:
				ec <- nil
				return
			}
		}
	}()
}

// Stop stops the EventsMonitor
func (em *EventsMonitor) Stop() {
	if em.stopChan == nil {
		return
	}
	close(em.stopChan)
}
