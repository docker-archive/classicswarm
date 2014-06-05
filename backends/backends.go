package backends

import (
	"fmt"
	"github.com/docker/libswarm/beam"
	"strings"
	"time"
)

// New returns a new engine, with all backends
// registered but not activated.
// To activate a backend, call a job on the resulting
// engine, named after the desired backend.
//
// Example: `New().Job("debug").Run()`
func New() *beam.Object {
	backends := beam.NewTree()
	backends.Bind("simulator", Simulator())
	backends.Bind("debug", Debug())
	backends.Bind("fakeclient", FakeClient())
	return beam.Obj(backends)
}

func Debug() beam.Sender {
	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
		instance := beam.Task(func(in beam.Receiver, out beam.Sender) {
			fmt.Printf("debug backend!")
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

func FakeClient() beam.Sender {
	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
		// Instantiate a new fakeclient instance
		instance := beam.Task(func(in beam.Receiver, out beam.Sender) {
			fmt.Printf("fake client!\n")
			defer fmt.Printf("end of fake client!\n")
			o := beam.Obj(out)
			o.Log("fake client starting")
			defer o.Log("fake client terminating")
			for {
				time.Sleep(1 * time.Second)
				o.Log("fake client heartbeat!")
			}
		})
		_, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: instance})
		return err
	}))
	return backend
}
