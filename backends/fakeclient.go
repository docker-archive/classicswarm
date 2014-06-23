package backends

import (
	"fmt"
	"time"

	"github.com/docker/libswarm"
	"github.com/docker/libswarm/utils"
)

func FakeClient() libswarm.Sender {
	backend := libswarm.NewServer()
	backend.OnVerb(libswarm.Spawn, libswarm.Handler(func(ctx *libswarm.Message) error {
		// Instantiate a new fakeclient instance
		instance := utils.Task(func(in libswarm.Receiver, out libswarm.Sender) {
			fmt.Printf("fake client!\n")
			defer fmt.Printf("end of fake client!\n")
			o := libswarm.Obj(out)
			o.Log("fake client starting")
			defer o.Log("fake client terminating")
			for {
				time.Sleep(1 * time.Second)
				o.Log("fake client heartbeat!")
			}
		})
		_, err := ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: instance})
		return err
	}))
	return backend
}

