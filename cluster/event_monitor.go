package cluster

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/swarm/swarmclient"
	"golang.org/x/net/context"
)

//EventsMonitor monitors events
type EventsMonitor struct {
	stopChan chan struct{}
	cli      swarmclient.SwarmAPIClient
	handler  func(msg events.Message) error
}

// NewEventsMonitor returns an EventsMonitor
func NewEventsMonitor(cli swarmclient.SwarmAPIClient, handler func(msg events.Message) error) *EventsMonitor {
	return &EventsMonitor{
		cli:     cli,
		handler: handler,
	}
}

// Start starts the EventsMonitor
func (em *EventsMonitor) Start(ec chan error) {
	em.stopChan = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	responseStream, errStream := em.cli.Events(ctx, types.EventsOptions{})

	go func() {
		defer cancel()
		for {
			select {
			case event := <-responseStream:
				if err := em.handler(event); err != nil {
					ec <- err
					return
				}
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
