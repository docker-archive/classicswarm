package cluster

import (
	"encoding/json"
	//"fmt"
	//"io"
	"time"

	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/events"
	"github.com/docker/swarm/swarmclient"
	"golang.org/x/net/context"
)

//EventsMonitor monitors events
type EventsMonitor struct {
	stopChan chan struct{}
	cli      swarmclient.SwarmAPIClient
	errChan  chan error
	handler  func(msg events.Message) error
	engine   *Engine
}

type decodingResult struct {
	msg events.Message
	err error
}

// NewEventsMonitor returns an EventsMonitor
//func NewEventsMonitor(engine *Engine, cli swarmclient.SwarmAPIClient, handler func(msg events.Message) error) *EventsMonitor {
func NewEventsMonitor(engine *Engine) *EventsMonitor {
	return &EventsMonitor{
		stopChan: make(chan struct{}, 2),
		errChan:  make(chan error),
		cli:      engine.apiClient,
		handler:  engine.handler,
		engine:   engine,
	}
}

// Start starts the EventsMonitor
func (em *EventsMonitor) Start() {
	//fmt.Println("Start EventMonitor.Start")
	go func() {
		for {
			select {
			case err := <-em.errChan:
				if err != nil {
					//fmt.Printf("----------- Event Stream ran into error: %v\n", err)
					// Swarm needs to wait an interval to start eventsMonitor.
					// If the Engine is lost forever, before Swarm judges it as disconnected，
					// Swarm will enter a loop to start eventsMonitor which may consume CPU a lot.
					time.Sleep(1 * time.Second)

					// EventMonitor ran into an error, start it again.
					go em.startEventsMonitoring()

				} else {
					// if err is nil, it means stopChan is closed and EventMonitor should return as well.
					return
				}
			case <-em.stopChan:
				//fmt.Println("em.StopChan has data, exiting goroutine in Start()")
				// if err is nil, it means stopChan is closed and EventMonitor should return as well.
				return
			}
		}
	}()

	// use a goroutine to make Start() return
	go em.startEventsMonitoring()
}

// startEventsMonitoring starts listening event stream of docker engine.
func (em *EventsMonitor) startEventsMonitoring() {
	//fmt.Println("-----------Start startEventsMonitoring")
	responseBody, err := em.cli.Events(context.Background(), types.EventsOptions{})
	em.engine.CheckConnectionErr(err)
	if err != nil {
		//fmt.Println("em.cli.Events error:" + err.Error())
		em.errChan <- err
		return
	}

	//resultChan := make(chan decodingResult)

	go func() {
		defer responseBody.Close()
		//fmt.Println("Enter in response processing")
		dec := json.NewDecoder(responseBody)

		for {
			select {
			case <-em.stopChan:
				//fmt.Println("Stop is called, Leaving response processing")
				return
			default:
				var result decodingResult
				result.err = dec.Decode(&result.msg)
				if result.err != nil {
					//fmt.Println("result.err != nil: " + result.err.Error())
					em.errChan <- result.err
					//fmt.Println("Leaving response processing.")
					return
				}
				if err := em.handler(result.msg); err != nil {
					//fmt.Println("em.handler err != nil: " + err.Error())
					em.errChan <- err
					//fmt.Println("Leaving response processing.")
					return
				}
			}
		}

		//fmt.Println("Leaving response processing.")
	}()
}

// Stop stops the EventsMonitor
func (em *EventsMonitor) Stop() {
	if em.stopChan == nil {
		//fmt.Println("-----------stopChan is nil")
		// channel stopChan is closed
		return
	}

	if len(em.stopChan) == 1 {
		//fmt.Println("-----------stopChan has data, is full.")
		// channel stopChan has data
		em.stopChan <- struct{}{}
		return
	}
	//fmt.Println("-----------stopChan is closed in Stop()")
	// nothing in the channel stopChan
	em.stopChan <- struct{}{}
	em.stopChan <- struct{}{}
}
