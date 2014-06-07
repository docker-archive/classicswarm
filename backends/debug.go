package backends

import (
	"fmt"
	"strings"

	"github.com/docker/libswarm/beam"
)

func Debug() beam.Sender {
	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
		instance := beam.Task(func(in beam.Receiver, out beam.Sender) {
			for {
				msg, err := in.Receive(beam.Ret)
				if err != nil {
					fmt.Printf("debug receive: %v", err)
					return
				}
				fmt.Printf("[DEBUG] %s %s\n", msg.Verb, strings.Join(msg.Args, " "))
				if _, err := out.Send(msg); err != nil {
					fmt.Printf("debug send: %v", err)
					return
				}
			}
		})
		_, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: instance})
		return err
	}))
	return backend
}
