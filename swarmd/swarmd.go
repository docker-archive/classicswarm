package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/docker/libswarm/backends"
	"github.com/docker/libswarm/beam"
	beamutils "github.com/docker/libswarm/beam/utils"
	_ "github.com/dotcloud/docker/api/server"
	"github.com/flynn/go-shlex"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	app := cli.NewApp()
	app.Name = "swarmd"
	app.Usage = "Control a heterogenous distributed system with the Docker API"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{"backend", "debug", "load a backend"},
	}
	app.Action = cmdDaemon
	app.Run(os.Args)
}

func cmdDaemon(c *cli.Context) {
	if len(c.Args()) == 0 {
		Fatalf("Usage: %s <proto>://<address> [<proto>://<address>]...\n", c.App.Name)
	}

	hub := beamutils.NewHub()
	hub.RegisterName("log", func(msg *beam.Message, out beam.Sender) (bool, error) {
		log.Printf("%s\n", strings.Join(msg.Args, " "))
		// Pass through to other logging hooks
		return true, nil
	})
	back := backends.New()
	// Load backends
	for _, cmd := range c.Args() {
		bName, bArgs, err := parseCmd(cmd)
		if err != nil {
			Fatalf("%v", err)
		}
		fmt.Printf("---> Loading backend '%s'\n", strings.Join(append([]string{bName}, bArgs...), " "))
		backend, err := back.Send(&beam.Message{Name: "cd", Args: []string{bName}, Ret: beam.RetPipe})
		if err != nil {
			Fatalf("%s: %v\n", bName, err)
		}
		// backend will return either 'error' or 'register'.
		for {
			m, err := backend.Receive(beam.Ret)
			if err == io.EOF {
				break
			}
			if err != nil {
				Fatalf("error reading from backend: %v", err)
			}
			if _, err := hub.Send(m); err != nil {
				Fatalf("error binding backend to hub: %v", err)
			}
		}
	}
	fmt.Printf("backends loaded. Sending 'start' to the hub\n")
	job, err := hub.Send(&beam.Message{Name: "start", Ret: beam.RetPipe})
	if err != nil {
		Fatalf("%v", err)
	}
	for {
		msg, err := job.Receive(0)
		if err == io.EOF {
			break
		}
		if err != nil {
			Fatalf("%v", err)
		}
		fmt.Printf("--> %s %s\n", msg.Name, strings.Join(msg.Args, " "))
	}
}

func parseCmd(txt string) (string, []string, error) {
	l, err := shlex.NewLexer(strings.NewReader(txt))
	if err != nil {
		return "", nil, err
	}
	var cmd []string
	for {
		word, err := l.NextWord()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, err
		}
		cmd = append(cmd, word)
	}
	if len(cmd) == 0 {
		return "", nil, fmt.Errorf("parse error: empty command")
	}
	return cmd[0], cmd[1:], nil
}

func Fatalf(msg string, args ...interface{}) {
	if !strings.HasSuffix(msg, "\n") {
		msg = msg + "\n"
	}
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}
