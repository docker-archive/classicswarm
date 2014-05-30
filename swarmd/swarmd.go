package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/docker/libswarm/backends"
	"github.com/dotcloud/docker/api/server"
	"github.com/dotcloud/docker/engine"
	"github.com/flynn/go-shlex"
	"io"
	"os"
	"strings"
)

func main() {
	app := cli.NewApp()
	app.Name = "swarmd"
	app.Usage = "Control a heterogenous distributed system with the Docker API"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{"backends", "debug", "load a backend"},
	}
	app.Action = cmdDaemon
	app.Run(os.Args)
}

func cmdDaemon(c *cli.Context) {
	if len(c.Args()) == 0 {
		Fatalf("Usage: %s <proto>://<address> [<proto>://<address>]...\n", c.App.Name)
	}

	// Load backends
	// Backends are defined as commands delimited by semi-colons
	back := backends.NewMux()
	backendCommands := strings.Split(c.String("backends"), ";")

	// Backends are treated as jobs that are started up in addition to
	// adding the backend to the engine multiplexer.
	for _, bCmd := range backendCommands {
		if bName, bArgs, err := parseCmd(strings.TrimSpace(bCmd)); err == nil {
			// Enable the backend engine in the engine multiplexer
			if err := back.Enable(bName, bArgs...); err != nil {
				Fatalf("Failed to load %s: %v\n", bName, err)
			}
		} else {
			Fatalf("Failed to parse command: %s, %v\n", bCmd, err)
		}
	}

	// Register the API entrypoint
	// (we register it as `argv[0]` so we can print usage messages straight from the job
	// stderr.
	front := engine.New()
	front.Logging = false

	// FIXME: server should expose an engine.Installer
	front.Register(c.App.Name, server.ServeApi)
	front.Register("acceptconnections", server.AcceptConnections)

	// Install the backend mux into the frontend
	back.Install(front)

	// Call the API entrypoint
	go func() {
		serve := front.Job(c.App.Name, c.Args()...)
		serve.Stdout.Add(os.Stdout)
		serve.Stderr.Add(os.Stderr)
		if err := serve.Run(); err != nil {
			Fatalf("serveapi: %v", err)
		}
	}()
	// Notify that we're ready to receive connections
	if err := front.Job("acceptconnections").Run(); err != nil {
		Fatalf("acceptconnections: %v", err)
	}
	// Inifinite loop
	<-make(chan struct{})
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
