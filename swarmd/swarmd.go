package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/docker/libswarm/backends"
	"github.com/docker/libswarm/beam"
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
	app := beam.NewServer()
	app.OnLog(beam.Handler(func(msg *beam.Message) error {
		log.Printf("%s\n", strings.Join(msg.Args, " "))
		return nil
	}))
	app.OnError(beam.Handler(func(msg *beam.Message) error {
		Fatalf("Fatal: %v", strings.Join(msg.Args[:1], ""))
		return nil
	}))
	back := backends.New()
	if len(c.Args()) == 0 {
		names, err := back.Ls()
		if err != nil {
			Fatalf("ls: %v", err)
		}
		fmt.Println(strings.Join(names, "\n"))
		return
	}
	bName, bArgs, err := parseCmd(c.Args()[0])
	if err != nil {
		Fatalf("parse: %v", err)
	}
	fmt.Printf("---> Loading backend '%s'\n", strings.Join(append([]string{bName}, bArgs...), " "))
	events, backend, err := back.Attach(bName)
	if err != nil {
		Fatalf("%s: %v\n", bName, err)
	}
	instance, err := backend.Spawn(bArgs...)
	if err != nil {
		Fatalf("spawn %s: %v\n", bName, err)
	}
	if err != nil {
		Fatalf("attach: %v", err)
	}
	if err := instance.Start(); err != nil {
		Fatalf("start: %v", err)
	}
	beam.Copy(app, events)
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
