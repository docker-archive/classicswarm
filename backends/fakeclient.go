package backends

import (
	"fmt"
	"time"

	"github.com/docker/libswarm/beam"
)

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

