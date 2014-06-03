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
	backends.Bind("debug", Debug())
	backends.Bind("fakeclient", FakeClient())
	return beam.Obj(backends)
}

func Debug() beam.Sender {
	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
		instance := beam.NewServer()
		instance.Catchall(beam.Handler(func(msg *beam.Message) error {
			fmt.Printf("[DEBUG] %s %s\n", msg.Name, strings.Join(msg.Args, " "))
			ctx.Ret.Send(msg)
			return nil
		}))
		_, err := ctx.Ret.Send(&beam.Message{Name: string(beam.Ack), Ret: instance})
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
		_, err := ctx.Ret.Send(&beam.Message{Name: string(beam.Ack), Ret: instance})
		return err
	}))
	return backend
}
