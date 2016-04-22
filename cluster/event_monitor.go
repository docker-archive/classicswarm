package cluster

import (
	"encoding/json"
	"io"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/events"
	"golang.org/x/net/context"
)

//EventsMonitor monitors events
type EventsMonitor struct {
	stopChan chan struct{}
	cli      client.APIClient
	handler  func(msg events.Message) error
}

type decodingResult struct {
	msg events.Message
	err error
}

// NewEventsMonitor returns an EventsMonitor
func NewEventsMonitor(cli client.APIClient, handler func(msg events.Message) error) *EventsMonitor {
	return &EventsMonitor{
		cli:     cli,
		handler: handler,
	}
}

// Start starts the EventsMonitor
func (em *EventsMonitor) Start(ec chan error) {
	em.stopChan = make(chan struct{})

	responseBody, err := em.cli.Events(context.TODO(), types.EventsOptions{})
	if err != nil {
		ec <- err
		return
	}

	resultChan := make(chan decodingResult)

	go func() {
		dec := json.NewDecoder(responseBody)
		for {
			var result decodingResult
			result.err = dec.Decode(&result.msg)
			resultChan <- result
			if result.err == io.EOF {
				break
			}
		}
		close(resultChan)
	}()

	go func() {
		defer responseBody.Close()
		for {
			select {
			case <-em.stopChan:
				ec <- nil
				return
			case result := <-resultChan:
				if result.err != nil {
					ec <- result.err
					return
				}
				if err := em.handler(result.msg); err != nil {
					ec <- err
					return
				}
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
