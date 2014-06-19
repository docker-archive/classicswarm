package debug

import (
	"fmt"
	"log"

	"github.com/docker/libswarm/beam"
)

// The Debug service is an example of intercepting messages between a receiver and a sender.
// The service also exposes messages passing through it for debug purposes.
func Debug() beam.Sender {
	dbgInstance := &debug{
		service: beam.NewServer(),
	}

	sender := beam.NewServer()
	sender.OnSpawn(beam.Handler(dbgInstance.spawn))
	return sender
}

// Function for forwarding a messgae including some minor error formatting
func forward(out beam.Sender, msg *beam.Message) (err error) {
	if _, err := out.Send(msg); err != nil {
		return fmt.Errorf("[debug] Failed to forward msg. Reason: %v\n", err)
	}
	return
}

// Debug service type
type debug struct {
	service *beam.Server
	in      beam.Receiver
	out     beam.Sender
}

// Spawn will return a new instance as the Ret channel of the message sent back
func (dbg *debug) spawn(msg *beam.Message) (err error) {
	// By sending back a task, beam will run the function with the in and out arguments
	// set to the services present before and after this one in the pipeline.
	instance := beam.Task(func(in beam.Receiver, out beam.Sender) {
		// Setup our channels
		dbg.in = in
		dbg.out = out

		// Set up the debug interceptor
		dbg.service.Catchall(beam.Handler(dbg.catchall))

		// Copy everything from the receiver to our service. By copying like this in the task
		// we can use the catchall handler instead of handling the message here.
		beam.Copy(dbg.service, in)
	})

	// Inform the system of our new instance
	msg.Ret.Send(&beam.Message{
		Verb: beam.Ack,
		Ret:  instance,
	})

	return
}

// Catches all messages sent to the service
func (dbg *debug) catchall(msg *beam.Message) (err error) {
	log.Printf("[debug] ---> Upstream Message { Verb: %s, Args: %v }\n", msg.Verb, msg.Args)

	// If there's no output after us then we'll just reply with an error
	// informing the receiver that the verb is not implemented.
	if dbg.out == nil {
		return fmt.Errorf("[debug] Verb: %s is not implemented.", msg.Verb)
	}

	// The forwarded message has the return channel set to a new replyHandler. The replyHandler is a small
	// callback that allows for interception of downstream messages.
	forwardedMessage := &beam.Message{
		Verb: msg.Verb,
		Args: msg.Args,
		Att:  msg.Att,
		Ret: &replyHandler{
			Sender: msg.Ret,
		},
	}

	// Send the forwarded message
	if err := forward(dbg.out, forwardedMessage); err == nil {
		// Hijack the return channel so we can avoid any interference with things such as close
		msg.Ret = beam.NopSender{}
	}

	return
}

// We use a replyHandler to provide context for relaying the return channel
// of the origin message.
type replyHandler struct {
	beam.Sender
}

// Send a message using the out channel
func (rh *replyHandler) Send(msg *beam.Message) (receiver beam.Receiver, err error) {
	log.Printf("[debug] <--- Downstream Message { Verb: %s, Args: %v }\n", msg.Verb, msg.Args)
	return nil, forward(rh.Sender, msg)
}

func (rh *replyHandler) Close() (err error) {
	// Since we don't allow the downstream handler to close the return channel, we do so here.
	return rh.Sender.Close()
}
