package auth

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/docker/libswarm/beam"
)

type authService struct {
	server    *beam.Server
	providers map[string]Provider
}

const (
	AUTH_USERNAME = "auth-user"
	AUTH_PASSWORD = "auth-password"
	AUTH_KEY      = "auth-key"
)

var authProviders map[string]Provider

func init() {
	authProviders = make(map[string]Provider)
}

func Authenticator() (sender beam.Sender) {
	as := &authService{
		providers: authProviders,
	}

	// Bind our spawn function
	as.server = beam.NewServer()
	as.server.OnSpawn(beam.Handler(as.spawn))
	return as.server
}

func (as *authService) providerNames() (providerNames []string) {
	providerNames = make([]string, 0, len(as.providers))
	for name, _ := range as.providers {
		providerNames = append(providerNames, name)
	}

	sort.Strings(providerNames)
	return
}

const (
	PROVIDERS_HEADER = "\nAVAILABLE AUTHENTICATION PROVIDERS:\n   "
)

func (as *authService) spawn(msg *beam.Message) (err error) {
	app := &cli.App{
		Name:    "auth",
		Usage:   "Authenticate messages using a variety of identity providers.",
		Version: "0.0.1",
		Flags: []cli.Flag{
			cli.StringFlag{"provider", "", "Sets the identity provider to authenticate against."},
		},
	}

	var provider Provider
	app.Action = func(ctx *cli.Context) {
		providerName := ctx.String("provider")

		if providerName == "" || providerName == "help" {
			app.Run([]string{"auth", "help"})
			err = fmt.Errorf("")
		} else if target, found := as.providers[providerName]; !found {
			err = fmt.Errorf("No auth provider: \"%s\"", providerName)
		} else {
			provider = target
		}
	}

	// Set up the args and then run them against the cli tool
	appArgs := []string{string(msg.Verb)}
	appErr := app.Run(append(appArgs, msg.Args...))
	if err == nil && appErr != nil {
		err = appErr
	}

	if err == nil {
		instance := beam.Task(func(in beam.Receiver, out beam.Sender) {
			// Keep a reference to where we should write messages to
			auth := &authenticator{}
			auth.out = out
			auth.provider = provider

			// Set up the authenticator server
			service := beam.NewServer()
			service.Catchall(beam.Handler(auth.catchall))

			// Copy everything from the receiver to our service
			beam.Copy(service, in)
		})

		// Inform the system of our new instance
		msg.Ret.Send(&beam.Message{
			Verb: beam.Ack,
			Ret:  instance,
		})
	} else {
		providerList := PROVIDERS_HEADER + strings.Join(as.providerNames(), "\n   ")
		fmt.Printf(providerList + "\n\n")
	}

	return
}

type Provider func(args map[string]string) (err error)

type authenticator struct {
	provider Provider
	out      beam.Sender
}

func (auth *authenticator) catchall(msg *beam.Message) (err error) {
	var (
		authEnv   map[string]string = make(map[string]string)
		numArgs   int               = len(msg.Args)
		cleanArgs []string          = make([]string, 0)
	)

	// Build the auth environment
	for idx := 0; idx < numArgs; idx++ {
		arg := msg.Args[idx]

		if !strings.HasPrefix(arg, "--auth-") {
			cleanArgs = append(cleanArgs, arg)
		} else {
			idx++
			if idx < numArgs {
				envKey := arg[2:]
				authEnv[envKey] = msg.Args[idx]
			}
		}
	}

	if err = auth.provider(authEnv); err == nil {
		for key, value := range authEnv {
			cleanArgs = append(cleanArgs, "--"+key)
			cleanArgs = append(cleanArgs, value)
		}

		// The auth environment will be added to the args after authentication
		// to allow for providers to set their own upstream environment
		forwardedMsg := &beam.Message{
			Verb: msg.Verb,
			Args: cleanArgs,
			Att:  msg.Att,
			Ret:  beam.RetPipe,
		}

		// Send the forwarded message
		var inbound beam.Receiver
		if inbound, err = auth.out.Send(forwardedMsg); err == nil {
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
					return fmt.Errorf("[debug] Failed to forward msg. Reason: %v\n", err)
				}
			}
		}
	}

	return
}
