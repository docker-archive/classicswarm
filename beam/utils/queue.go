package utils

import (
	"github.com/docker/libswarm/beam"
	"github.com/docker/libswarm/beam/inmem"
)

type Queue struct {
	*inmem.PipeSender
	dst beam.Sender
	ch  chan *beam.Message
}

func NewQueue(dst beam.Sender, size int) *Queue {
	r, w := inmem.Pipe()
	q := &Queue{
		PipeSender: w,
		dst:        dst,
		ch:         make(chan *beam.Message, size),
	}
	go func() {
		defer close(q.ch)
		for {
			msg, err := r.Receive(beam.Ret)
			if err != nil {
				r.Close()
				return
			}
			q.ch <- msg
		}
	}()
	go func() {
		for msg := range q.ch {
			_, err := dst.Send(msg)
			if err != nil {
				r.Close()
				return
			}
		}
	}()
	return q
}
