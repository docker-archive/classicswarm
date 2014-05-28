package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/docker/libswarm/backends"
	"github.com/dotcloud/docker/api/server"
	"github.com/dotcloud/docker/engine"
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

	// Register any backends
	back := backends.New()
	if cmd := c.String("backends"); cmd != "" {
		if err := back.Job("backends", cmd).Run(); err != nil {
			Fatalf("Failed to init backends. Reason: %v.", err)
			return
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
	backends.Link(front, back)

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

func Fatalf(msg string, args ...interface{}) {
	if !strings.HasSuffix(msg, "\n") {
		msg = msg + "\n"
	}
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}
