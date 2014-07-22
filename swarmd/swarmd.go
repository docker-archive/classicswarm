package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/docker/libswarm"
	"github.com/docker/libswarm/backends"
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
	app.Flags = []cli.Flag{}
	app.Action = cmdDaemon
	app.Run(os.Args)
}

func cmdDaemon(c *cli.Context) {
	app := libswarm.NewServer()
	app.OnLog(func(args ...string) error {
		log.Printf("%s\n", strings.Join(args, " "))
		return nil
	})
	app.OnError(func(args ...string) error {
		Fatalf("Fatal: %v", strings.Join(args[:1], ""))
		return nil
	})
	back := backends.New()
	if len(c.Args()) == 0 {
		names, err := back.Ls()
		if err != nil {
			Fatalf("ls: %v", err)
		}
		fmt.Println(strings.Join(names, "\n"))
		return
	}
	var previousInstanceR libswarm.Receiver
	// FIXME: refactor into a Pipeline
	for idx, backendArg := range c.Args() {
		bName, bArgs, err := parseCmd(backendArg)
		if err != nil {
			Fatalf("parse: %v", err)
		}
		_, backend, err := back.Attach(bName)
		if err != nil {
			Fatalf("%s: %v\n", bName, err)
		}
		instance, err := backend.Spawn(bArgs...)
		if err != nil {
			Fatalf("spawn %s: %v\n", bName, err)
		}
		instanceR, instanceW, err := instance.Attach("")
		if err != nil {
			Fatalf("attach: %v", err)
		}
		go func(r libswarm.Receiver, w libswarm.Sender, idx int) {
			if r != nil {
				libswarm.Copy(w, r)
			}
			w.Close()
		}(previousInstanceR, instanceW, idx)
		if err := instance.Start(); err != nil {
			Fatalf("start: %v", err)
		}
		previousInstanceR = instanceR
	}
	_, err := libswarm.Copy(app, previousInstanceR)
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
