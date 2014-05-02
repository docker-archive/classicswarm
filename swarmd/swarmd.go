package main

import (
	"fmt"
	"github.com/docker/swarmd/backends"
	"github.com/dotcloud/docker/api/server"
	"github.com/dotcloud/docker/engine"
	"os"
	"time"
)

func main() {
	eng := engine.New()
	eng.Logging = false
	if err := backends.Debug().Install(eng); err != nil {
		Fatalf("backend install: %v", err)
	}
	eng.Register(os.Args[0], server.ServeApi)

	// Register the entrypoint job as the current proces command name,
	// to get matching usage straight from the job.
	go func() {
		serve := eng.Job(os.Args[0], os.Args[1:]...)
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
