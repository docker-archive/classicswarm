package cluster

import (
	"io"

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
	responseStream, errStream := em.cli.Events(context.Background(), types.EventsOptions{})

	go func() {
		for {
			select {
			case event := <-responseStream:
				if err := em.handler(event); err != nil {
					ec <- err
					return
				}
			case err := <-errStream:
				if err == io.EOF {
					ec <- nil
					return
				}
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
