package utils

import (
	"fmt"
	"github.com/docker/beam"
	"github.com/docker/beam/inmem"
	"sync"
)

// Hub passes messages to dynamically registered handlers.
type Hub struct {
	handlers *StackSender
	tasks    sync.WaitGroup
}

func NewHub() *Hub {
	return &Hub{
		handlers: NewStackSender(),
	}
}

func (hub *Hub) Send(msg *beam.Message, mode int) (beam.Receiver, beam.Sender, error) {
	if msg.Name == "register" {
		if mode&beam.R == 0 {
			return nil, nil, fmt.Errorf("register: no return channel")
		}
		fmt.Printf("[hub] received %v\n", msg)
		hYoutr, hYoutw := inmem.Pipe()
		hYinr, hYinw := inmem.Pipe()
		// Register the new handler on top of the others,
		// and get a reference to the previous stack of handlers.
		prevHandlers := hub.handlers.Add(hYinw)
		// Pass requests from the new handler to the previous chain of handlers
		// hYout -> hXin
		hub.tasks.Add(1)
		go func() {
			defer hub.tasks.Done()
			Copy(prevHandlers, hYoutr)
			hYoutr.Close()
		}()
		return hYinr, hYoutw, nil
	}
	fmt.Printf("sending %#v to %d handlers\n", msg, hub.handlers.Len())
	return hub.handlers.Send(msg, mode)
}

func (hub *Hub) Wait() {
	hub.tasks.Wait()
}

func (hub *Hub) Close() error {
	return hub.handlers.Close()
}
