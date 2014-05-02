package main

import (
	"fmt"
	"github.com/docker/swarmd/backends"
	"github.com/dotcloud/docker/api/server"
	"github.com/dotcloud/docker/engine"
	"os"
	"time"
	"github.com/codegangsta/cli"
	"strings"
)

func main() {
	app := cli.NewApp()
	app.Name = "swarmd"
	app.Usage = "Control a heterogenous distributed system with the Docker API"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
	}
	app.Action = cmdDaemon
	app.Run(os.Args)
}

func cmdDaemon(c *cli.Context) {
	if len(c.Args()) == 0 {
		Fatalf("Usage: %s <proto>://<address> [<proto>://<address>]...\n", c.App.Name)
	}
	eng := engine.New()
	eng.Logging = false
	if err := backends.Debug().Install(eng); err != nil {
		Fatalf("backend install: %v", err)
	}

	// Register the API entrypoint
	// (we register it as `argv[0]` so we can print usage messages straight from the job
	// stderr.
	eng.Register(c.App.Name, server.ServeApi)

	// Call the API entrypoint
	go func() {
		serve := eng.Job(c.App.Name, c.Args()...)
		serve.Stdout.Add(os.Stdout)
		serve.Stderr.Add(os.Stderr)
		if err := serve.Run(); err != nil {
			Fatalf("serveapi: %v", err)
		}
	}()
	// There is a race condition in engine.ServeApi.
	// As a workaround we sleep to give it time to register 'acceptconnections'.
	time.Sleep(1 * time.Second)
	// Notify that we're ready to receive connections
	if err := eng.Job("acceptconnections").Run(); err != nil {
		Fatalf("acceptconnections: %v", err)
	}
	// Inifinite loop
	<-make(chan struct{})
}

func Fatalf(msg string, args ...interface{}) {
	if !strings.HasSuffix(msg, "\n") {
		msg = msg + "\n"
	}
	panic(msg)
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}
