package backends

import (
	"io"
	"fmt"
	"reflect"
	"github.com/docker/libswarm/beam"
)

func Interceptor() (beam.Sender) {
	spawn :=  func (msg *beam.Message) (err error) {
		instance := beam.Task(func (in beam.Receiver, out beam.Sender) {
			service := beam.NewServer()

			ai := &interceptor {
				service: service,
				in: in,
				out: out,
			}

			// Set up the interceptor server
			service.Catchall(beam.Handler(ai.catchall))

			// Copy everything from the receiver to our service
			beam.Copy(service, in)
		})

		// Inform the system of our new instance
		msg.Ret.Send(&beam.Message {
			Verb: beam.Ack,
			Ret: instance,
		})

		return
	}

	sender := beam.NewServer()
	sender.OnSpawn(beam.Handler(spawn))
	return sender
}

type interceptor struct {
	service *beam.Server
	in beam.Receiver
	out beam.Sender
}

func (ai *interceptor) catchall(msg *beam.Message) (err error) {
	beam.Obj(ai.out).Log("[interceptor] Caught msg --> Verb: %s, Args: %v, Reciever: %v, Sender: %v\n", msg.Verb, msg.Args, reflect.TypeOf(msg.Ret), reflect.TypeOf(ai.out))

	// The forwarded message requests from beam that a pipe be created. This
	// is done so we can retrieve any results send back across the return.
	forwardedMessage := &beam.Message{
		Verb: msg.Verb,
		Args: msg.Args,
		Att: msg.Att,
		Ret: beam.RetPipe,
	}

	// Send the forwarded message
	if inbound, err := ai.out.Send(forwardedMessage); err != nil {
		return err
	} else {
		// Get any replies sent back across the pipe
		reply, err := inbound.Receive(0)

		if err == io.EOF {
			return fmt.Errorf("[interceptor] Unexpected EOF in reply from upstream.")
		}

		if _, err = msg.Ret.Send(reply); err != nil {
			return fmt.Errorf("[interceptor] Failed to forward msg! Verb: %s, Args: %v\n", reply.Verb, reply.Args)
		}
	}
	return
}
