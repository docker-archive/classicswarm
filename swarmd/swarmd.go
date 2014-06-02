package main

import (
	"bufio"
	"fmt"
	"log"
	"github.com/codegangsta/cli"
	"github.com/docker/libswarm/beam"
	"github.com/docker/libswarm/beam/inmem"
	beamutils "github.com/docker/libswarm/beam/utils"
	"github.com/docker/libswarm/backends"
	_ "github.com/dotcloud/docker/api/server"
	"github.com/dotcloud/docker/engine"
	"github.com/flynn/go-shlex"
	"io"
	"os"
	"strings"
	"sync"
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

func EngineAsSender(eng *engine.Engine) beam.Sender {
	r, w := inmem.Pipe()
	go func() {
		for {
			msg, msgr, msgw, err := r.Receive(beam.R | beam.W)
			if err != nil {
				return
			}
			go func(msg *beam.Message, in beam.Receiver, out beam.Sender) {
				job := eng.Job(msg.Name, msg.Args...)
				stdout, _ := job.Stdout.AddPipe() // can't fail
				stderr, _ := job.Stderr.AddPipe() // can't fail
				stdinR, stdinW := io.Pipe()
				defer stdinR.Close()
				defer stdinW.Close()
				job.Stdin.Add(stdinR)
				log := func(src io.Reader) {
					scanner := bufio.NewScanner(src)
					for scanner.Scan() {
						if scanner.Err() != nil {
							return
						}
						if _, _, err := out.Send(&beam.Message{Name: "log", Args: []string{scanner.Text()}}, 0); err != nil {
							return
						}
					}
				}
				var tasks sync.WaitGroup
				tasks.Add(3)
				go func() {
					// Read from stdout, send "log" events
					defer tasks.Done()
					log(stdout)
				}()
				go func() {
					// Read from stderr, send "log" events
					// FIXME: how to differentiate stderr/stdout logs?
					defer tasks.Done()
					log(stderr)
				}()
				go func() {
					// Receive events, send "log" events to stdin
					defer tasks.Done()
					for {
						m, _, _, err := in.Receive(0)
						if err != nil {
							return
						}
						if m.Name == "log" {
							if len(m.Args) < 1 {
								continue
							}
							fmt.Fprintf(stdinW, "%s\n", strings.TrimRight(m.Args[0], "\r\n"))
						}
					}
				}()
				err := job.Run()
				if err != nil {
					out.Send(&beam.Message{Name: "error", Args: []string{err.Error()}}, 0)
				}
			}(msg, msgr, msgw)
		}
	}()
	return w
}

func SenderAsEngine(s beam.Sender) *engine.Engine {
	eng := engine.New()
	eng.RegisterCatchall(func(job *engine.Job) engine.Status {
		msg := &beam.Message{
			Name: job.Name,
			Args: job.Args,
		}
		// FIXME: serialize job.Env into a trailing argument
		r, w, err := s.Send(msg, beam.R|beam.W)
		if err != nil {
			return job.Errorf("beam send: %v", err)
		}
		var tasks sync.WaitGroup
		tasks.Add(1)
		go func() {
			defer tasks.Done()
			in := bufio.NewScanner(job.Stdin)
			for in.Scan() {
				_, _, err := w.Send(&beam.Message{Name: "log", Args: []string{in.Text()}}, 0)
				if err != nil {
					return
				}
			}
		}()
		tasks.Add(1)
		var status engine.Status = engine.StatusOK
		go func() {
			defer tasks.Done()
			for {
				msg, _, _, err := r.Receive(0)
				if err != nil {
					return
				}
				if msg.Name == "log" {
					if len(msg.Args) < 1 {
						continue
					}
					fmt.Fprintf(job.Stdout, "%s\n", strings.TrimRight(msg.Args[0], "\r\n"))
				} else if msg.Name == "error" {
					status = engine.StatusErr
					if len(msg.Args) < 1 {
						continue
					}
					fmt.Fprintf(job.Stderr, "%s\n", strings.TrimRight(msg.Args[0], "\r\n"))
				}
			}
		}()
		tasks.Wait()
		return status
	})
	return eng
}

func cmdDaemon(c *cli.Context) {
	if len(c.Args()) == 0 {
		Fatalf("Usage: %s <proto>://<address> [<proto>://<address>]...\n", c.App.Name)
	}

	hub := beamutils.NewHub()
	hub.RegisterName("log", func(msg *beam.Message, in beam.Receiver, out, next beam.Sender) (bool, error) {
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
		backendr, _, err := back.Send(&beam.Message{Name: "cd", Args: []string{bName}}, beam.R)
		if err != nil {
			Fatalf("%s: %v\n", bName, err)
		}
		// backendr will return either 'error' or 'register'.
		for {
			m, mr, mw, err := backendr.Receive(beam.R|beam.W)
			if err == io.EOF {
				break
			}
			if err != nil {
				Fatalf("error reading from backend: %v", err)
			}
			if m.Name == "error" {
				Fatalf("backend sent error: %v", strings.Join(m.Args, " "))
			}
			if m.Name == "register" {
				// FIXME: adapt the beam interface to allow the caller to
				// (optionally) pass their own Sender/Receiver?
				// Would make proxying/splicing easier.
				hubr, hubw, err := hub.Send(m, beam.R|beam.W)
				if err != nil {
					Fatalf("error binding backend to hub: %v", err)
				}
				fmt.Printf("successfully registered\n")
				go beamutils.Copy(hubw, mr)
				go beamutils.Copy(mw, hubr)
			}
		}
	}
	in, _, err := hub.Send(&beam.Message{Name: "start"}, beam.R)
	if err != nil {
		Fatalf("%v", err)
	}
	for {
		msg, _, _, err := in.Receive(0)
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
