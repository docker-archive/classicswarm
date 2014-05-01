package main

import (
	"github.com/dotcloud/docker/api/server"
	"github.com/dotcloud/docker/engine"
	"strings"
	"os"
	"fmt"
	"time"
)

func main() {
	eng := engine.New()
	eng.Logging = false
	eng.RegisterCatchall(func(job *engine.Job) engine.Status {
		fmt.Printf("--> %s %s\n", job.Name, strings.Join(job.Args, " "))
		for k, v := range job.Env().Map() {
			fmt.Printf("        %s=%s\n", k, v)
		}
		// This helps us detect the race condition if our time.Sleep
		// missed it. (see comment below)
		if job.Name == "acceptconnections" {
			panic("race condition in github.com/dotcloud/docker/api/server/ServeApi")
		}
		return engine.StatusOK
	})
	eng.Register(os.Args[0], server.ServeApi)

	// Register the entrypoint job as the current proces command name,
	// to get matching usage straight from the job.
	go func() {
		serve := eng.Job(os.Args[0], os.Args[1:]...)
		serve.Stdout.Add(os.Stdout)
		serve.Stderr.Add(os.Stderr)
		if err := serve.Run(); err != nil {
			Fatalf("%v", err)
		}
	}()
	// There is a race condition in engine.ServeApi.
	// As a workaround we sleep to give it time to register 'acceptconnections'.
	time.Sleep(1)
	// Notify that we're ready to receive connections
	if err := eng.Job("acceptconnections").Run(); err != nil {
		Fatalf("%v", err)
	}
	// Inifinite loop
	<-make(chan struct{})
}

func Fatalf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}
