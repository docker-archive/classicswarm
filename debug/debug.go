package debug

import (
	"fmt"
	"io"
	"log"

	"github.com/docker/libswarm"
	"github.com/docker/libswarm/utils"
)

// The Debug service is an example of intercepting messages between a receiver and a sender.
// The service also exposes messages passing through it for debug purposes.
func Debug() libswarm.Sender {
	dbgInstance := &debug{
		service: libswarm.NewServer(),
	}

	sender := libswarm.NewServer()
	sender.OnVerb(libswarm.Spawn, libswarm.Handler(dbgInstance.spawn))
	return sender
}

// Debug service type
type debug struct {
	service *libswarm.Server
	out     libswarm.Sender
}

// Spawn will return a new instance as the Ret channel of the message sent back
func (dbg *debug) spawn(msg *libswarm.Message) (err error) {
	// By sending back a task, libswarm will run the function with the in and out arguments
	// set to the services present before and after this one in the pipeline.
	instance := utils.Task(func(in libswarm.Receiver, out libswarm.Sender) {
		// Setup our channels
		dbg.out = out

		// Set up the debug interceptor
		dbg.service.Catchall(libswarm.Handler(dbg.catchall))

		// Copy everything from the receiver to our service. By copying like this in the task
		// we can use the catchall handler instead of handling the message here.
		libswarm.Copy(dbg.service, in)
	})

	// Inform the system of our new instance
	msg.Ret.Send(&libswarm.Message{
		Verb: libswarm.Ack,
		Ret:  instance,
	})

	return
}

// Catches all messages sent to the service
func (dbg *debug) catchall(msg *libswarm.Message) (err error) {
	log.Printf("[debug] ---> Outbound Message ---> { Verb: %s, Args: %v }\n", msg.Verb, msg.Args)

	// If there's no output after us then we'll just reply with an error
	// informing the receiver that the verb is not implemented.
	if dbg.out == nil {
		return fmt.Errorf("[debug] Verb: %s is not implemented.", msg.Verb)
	}

	// We forward the message with a special Ret value of "libswarm.RetPipe" - this
	// asks libchan to open a new pipe so that we can read replies from upstream
	forwardedMsg := &libswarm.Message{
		Verb: msg.Verb,
		Args: msg.Args,
		Att:  msg.Att,
		Ret:  libswarm.RetPipe,
	}

	// Send the forwarded message
	if inbound, err := dbg.out.Send(forwardedMsg); err != nil {
		return fmt.Errorf("[debug] Failed to forward msg. Reason: %v\n", err)
	} else if inbound == nil {
		return fmt.Errorf("[debug] Inbound channel nil.\n")
	} else {
		for {
			// Relay all messages returned until the inbound channel is empty (EOF)
			var reply *libswarm.Message
			if reply, err = inbound.Receive(0); err != nil {
				if err == io.EOF {
					// EOF is expected
					err = nil
				}
				break
			}

			// Forward the message back downstream
			if _, err = msg.Ret.Send(reply); err != nil {
				return fmt.Errorf("[debug] Failed to forward msg. Reason: %v\n", err)
			}
			log.Printf("[debug] <--- Inbound Message <--- { Verb: %s, Args: %v }\n", reply.Verb, reply.Args)
		}
	}

	return
}
