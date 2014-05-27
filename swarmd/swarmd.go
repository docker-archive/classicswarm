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
		cli.StringFlag{"backend", "debug", "load a backend"},
	}
	app.Action = cmdDaemon
	app.Run(os.Args)
}

func cmdDaemon(c *cli.Context) {
	if len(c.Args()) == 0 {
		Fatalf("Usage: %s <proto>://<address> [<proto>://<address>]...\n", c.App.Name)
	}

	// Load backend
	// FIXME: allow for multiple backends to be loaded.
	// This could be done by instantiating 1 engine per backend,
	// installing each backend in its respective engine,
	// then registering a Catchall on the frontent engine which
	// multiplexes across all backends (with routing / filtering
	// logic along the way).
	back := backends.New()
	bName, bArgs, err := parseCmd(c.String("backend"))
	if err != nil {
		Fatalf("%v", err)
	}
	fmt.Printf("---> Loading backend '%s'\n", strings.Join(append([]string{bName}, bArgs...), " "))
	if err := back.Job(bName, bArgs...).Run(); err != nil {
		Fatalf("%s: %v\n", bName, err)
	}

	// Register the API entrypoint
	// (we register it as `argv[0]` so we can print usage messages straight from the job
	// stderr.
	front := engine.New()
	front.Logging = false
	// FIXME: server should expose an engine.Installer
	front.Register(c.App.Name, server.ServeApi)
	front.Register("acceptconnections", server.AcceptConnections)
	front.RegisterCatchall(func(job *engine.Job) engine.Status {
		fw := back.Job(job.Name, job.Args...)
		fw.Stdout.Add(job.Stdout)
		fw.Stderr.Add(job.Stderr)
		fw.Stdin.Add(job.Stdin)
		for key, val := range job.Env().Map() {
			fw.Setenv(key, val)
		}
		fw.Run()
		return engine.Status(fw.StatusCode())
	})

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
