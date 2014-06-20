package inject

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/docker/libswarm/beam"
)

const (
	USAGE = `Beam Argument Injector\n`
)

type injectorService struct {
	server *beam.Server
}

func Injector() (server *beam.Server) {
	server = beam.NewServer()
	argInjector := &injectorService{
		server: server,
	}

	// Bind our spawn function
	server.OnSpawn(beam.Handler(argInjector.spawn))
	return
}

func (is *injectorService) spawn(msg *beam.Message) (err error) {
	if len(msg.Args) == 0 {
		return fmt.Errorf(USAGE)
	}

	filteredArgs := make([]string, 0)
	for _, arg := range msg.Args {
		filteredArg := arg

		if len(arg) >= 2 {
			if strings.HasPrefix(arg, "\\@") {
				filteredArg = arg[1:]
			} else if strings.HasPrefix(arg, "@") {
				if contents, err := ioutil.ReadFile(arg[1:]); err != nil {
					return err
				} else {
					filteredArg = strings.Trim(string(contents), "\r\n\t ")
				}
			}
		}

		filteredArgs = append(filteredArgs, filteredArg)
	}

	instance := beam.Task(func(in beam.Receiver, out beam.Sender) {
		// Keep a reference to where we should write messages to
		i := &injector{}
		i.out = out
		i.args = filteredArgs

		// Set up the authenticator server
		service := beam.NewServer()
		service.Catchall(beam.Handler(i.catchall))

		// Copy everything from the receiver to our service
		beam.Copy(service, in)
	})

	// Inform the system of our new instance
	msg.Ret.Send(&beam.Message{
		Verb: beam.Ack,
		Ret:  instance,
	})

	return
}

type injector struct {
	out  beam.Sender
	args []string
}

// Catches all messages sent to the service
func (i *injector) catchall(msg *beam.Message) (err error) {
	forwardedMsg := &beam.Message{
		Verb: msg.Verb,
		Args: append(msg.Args, i.args...),
		Att:  msg.Att,
		Ret:  beam.RetPipe,
	}

	// Send the forwarded message
	var inbound beam.Receiver
	if inbound, err = i.out.Send(forwardedMsg); err == nil {
		for {
			// Relay all messages returned until the inbound channel is empty (EOF)
			var reply *beam.Message
			if reply, err = inbound.Receive(0); err != nil {
				if err == io.EOF {
					// EOF is expected
					err = nil
				}
				break
			}

			// Forward the message back downstream
			if _, err = msg.Ret.Send(reply); err != nil {
				return fmt.Errorf("[auth] Failed to forward msg. Reason: %v\n", err)
			}
		}
	}

	return
}
