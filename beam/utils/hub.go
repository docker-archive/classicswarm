package utils

import (
	"fmt"
	"github.com/docker/libswarm/beam"
	"io"
	"strings"
	"sync"
)

// Hub passes messages to dynamically registered handlers.
type Hub struct {
	handlers *StackSender
	tasks    sync.WaitGroup
	l        sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		handlers: NewStackSender(),
	}
}

func (hub *Hub) Send(msg *beam.Message) (ret beam.Receiver, err error) {
	if msg.Name == "register" {
		if msg.Ret == nil {
			return nil, fmt.Errorf("register: no return channel")
		}
		fmt.Printf("[hub] received %v\n", msg)
		hIn := msg.Ret
		if hIn == beam.RetPipe {
			ret, hIn = beam.Pipe()
		}
		// This queue guarantees that the first message received by the handler
		// is the "register" response.
		hIn = NewQueue(hIn, 1)
		// Reply to the handler with a "register" call of our own,
		// passing a reference to the previous handler stack.
		// This allows the new handler to query previous handlers
		// without creating loops.
		hOut, err := hIn.Send(&beam.Message{Name: "register", Ret: beam.RetPipe})
		if err != nil {
			return nil, err
		}
		// Register the new handler on top of the others,
		// and get a reference to the previous stack of handlers.
		prevHandlers := hub.handlers.Add(hIn)
		go beam.Copy(prevHandlers, hOut)
		return ret, nil
	}
	fmt.Printf("sending %#v to %d handlers\n", msg, hub.handlers.Len())
	return hub.handlers.Send(msg)
}

func (hub *Hub) RegisterTask(h func(beam.Receiver, beam.Sender) error) error {
	ret, err := hub.Send(&beam.Message{Name: "register", Ret: beam.RetPipe})
	if err != nil {
		return err
	}
	ack, err := ret.Receive(beam.Ret)
	if err != nil {
		return err
	}
	if ack.Name == "error" {
		return fmt.Errorf(strings.Join(ack.Args, ", "))
	}
	if ack.Name != "register" {
		return fmt.Errorf("invalid response: expected verb 'register', got '%v'", ack.Name)
	}
	go func() {
		h(ret, ack.Ret)
		ack.Ret.Close()
	}()
	return nil
}

type Handler func(msg *beam.Message, out beam.Sender) (pass bool, err error)

func (hub *Hub) RegisterName(name string, h Handler) error {
	return hub.RegisterTask(func(in beam.Receiver, out beam.Sender) error {
		for {
			msg, err := in.Receive(beam.Ret)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			var pass = true
			if msg.Name == name || name == "" {
				pass, err = h(msg, out)
				if err != nil {
					if _, err := msg.Ret.Send(&beam.Message{Name: "error", Args: []string{err.Error()}}); err != nil {
						return err
					}
				}
			}
			if pass {
				if _, err := out.Send(msg); err != nil {
					return err
				}
			} else {
				msg.Ret.Close()
			}
		}
		return nil
	})
}

func (hub *Hub) Wait() {
	hub.tasks.Wait()
}

func (hub *Hub) Close() error {
	return hub.handlers.Close()
}
