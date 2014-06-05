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
	app.Usage = "Compose distributed systems from lightweight services"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
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
	var previousInstanceIn beam.Receiver
	for _, backendArg := range c.Args() {
		bName, bArgs, err := parseCmd(backendArg)
		if err != nil {
			Fatalf("parse: %v", err)
		}
		fmt.Printf("---> Loading backend '%s'\n", strings.Join(append([]string{bName}, bArgs...), " "))
		_, backend, err := back.Attach(bName)
		if err != nil {
			Fatalf("%s: %v\n", bName, err)
		}
		fmt.Printf("---> Spawning\n")
		instance, err := backend.Spawn(bArgs...)
		if err != nil {
			Fatalf("spawn %s: %v\n", bName, err)
		}
		fmt.Printf("---> Attaching\n")
		instanceIn, instanceOut, err := instance.Attach("")
		if err != nil {
			Fatalf("attach: %v", err)
		}
		fmt.Printf("---> Starting\n")
		if err := instance.Start(); err != nil {
			Fatalf("start: %v", err)
		}
		if previousInstanceIn != nil {
			go beam.Copy(instanceOut, previousInstanceIn)
		}
		previousInstanceIn = instanceIn
	}
	_, err := beam.Copy(app, previousInstanceIn)
	if err != nil {
		Fatalf("copy: %v", err)
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
