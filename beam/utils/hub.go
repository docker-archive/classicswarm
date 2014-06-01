package utils

import (
	"fmt"
	"github.com/docker/beam"
	"github.com/docker/beam/inmem"
	"io"
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

func (hub *Hub) Register(dst beam.Sender) error {
	in, _, err := hub.Send(&beam.Message{Name: "register"}, beam.R)
	if err != nil {
		return err
	}
	go Copy(dst, in)
	return nil
}

func (hub *Hub) RegisterTask(h func(beam.Receiver, beam.Sender) error) error {
	in, out, err := hub.Send(&beam.Message{Name: "register"}, beam.R|beam.W)
	if err != nil {
		return err
	}
	go func() {
		h(in, out)
		out.Close()
	}()
	return nil
}

type Handler func(msg *beam.Message, in beam.Receiver, out beam.Sender, next beam.Sender) (pass bool, err error)

func (hub *Hub) RegisterName(name string, h Handler) error {
	return hub.RegisterTask(func(in beam.Receiver, out beam.Sender) error {
		var tasks sync.WaitGroup
		copyTask := func(dst beam.Sender, src beam.Receiver) {
			tasks.Add(1)
			go func() {
				defer tasks.Done()
				if dst == nil {
					return
				}
				defer dst.Close()
				if src == nil {
					return
				}
				Copy(dst, src)
			}()
		}
		for {
			msg, msgin, msgout, err := in.Receive(beam.R | beam.W)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			var pass = true
			if msg.Name == name || name == "" {
				pass, err = h(msg, msgin, msgout, out)
				if err != nil {
					if _, _, err := msgout.Send(&beam.Message{Name: "error", Args: []string{err.Error()}}, 0); err != nil {
						return err
					}
				}
			}
			if pass {
				nextin, nextout, err := out.Send(msg, beam.R|beam.W)
				if err != nil {
					return err
				}
				copyTask(nextout, msgin)
				copyTask(msgout, nextin)
			} else {
				if msgout != nil {
					msgout.Close()
				}
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
