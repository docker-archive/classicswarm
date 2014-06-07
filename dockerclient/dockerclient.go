package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/docker/libswarm/backends"
	"github.com/docker/libswarm/beam"
	"github.com/dotcloud/docker/engine"
	"github.com/dotcloud/docker/runconfig"
	"github.com/dotcloud/docker/utils"
	"io"
	"os"
	"strings"
	"text/tabwriter"
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
		utils.Debugf("%s", strings.Join(msg.Args, " "))
		return nil
	}))
	app.OnError(beam.Handler(func(msg *beam.Message) error {
		Fatalf("Fatal: %v", strings.Join(msg.Args[:1], ""))
		return nil
	}))

	backend := beam.Object{backends.Forward()}

	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}

	instance, err := backend.Spawn(dockerHost)
	if err != nil {
		Fatalf("spawn: %v\n", err)
	}

	instanceR, instanceW, err := instance.Attach("")
	if err != nil {
		Fatalf("attach: %v", err)
	}
	defer instanceW.Close()
	go beam.Copy(app, instanceR)

	if err := instance.Start(); err != nil {
		Fatalf("start: %v", err)
	}

	err = doCmd(instance, c.Args())
	if err != nil {
		Fatalf("%v", err)
	}
}

func doCmd(instance *beam.Object, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command supplied")
	}
	if args[0] == "ps" {
		if len(args) != 1 {
			return fmt.Errorf("usage: ps")
		}
		containers, err := instance.GetChildren()
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stderr, 20, 1, 3, ' ', 0)
		fmt.Fprint(w, "CONTAINER ID\tIMAGE\tCOMMAND\tSTATUS\n")
		for _, envJson := range containers {
			var out engine.Env
			buffer := bytes.NewBufferString(envJson)
			err := out.Decode(buffer)
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				utils.TruncateID(out.Get("Id")),
				out.Get("Image"),
				utils.Trunc(out.Get("Command"), 20),
				out.Get("Status"),
			)
		}
		w.Flush()
		return nil
	}
	if args[0] == "run" {
		if len(args) < 3 {
			return fmt.Errorf("usage: run IMAGE COMMAND...")
		}
		containerJson, err := json.Marshal(&runconfig.Config{
			Image:        args[1],
			Cmd:          args[2:],
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
		})
		if err != nil {
			return err
		}
		container, err := instance.Spawn(string(containerJson))
		if err != nil {
			return fmt.Errorf("spawn: %v", err)
		}
		logs, _, err := container.Attach("")
		if err != nil {
			return fmt.Errorf("attach: %v", err)
		}
		if err = container.Start(); err != nil {
			return fmt.Errorf("start: %v", err)
		}
		for {
			msg, err := logs.Receive(beam.Ret)
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				return fmt.Errorf("error reading from container: %v", err)
			}
			if msg.Verb != beam.Log {
				return fmt.Errorf("unexpected message reading from container: %v", msg)
			}
			if len(msg.Args) != 2 {
				return fmt.Errorf("expected exactly 2 args to log message, got %d", len(msg.Args))
			}
			tag, chunk := msg.Args[0], msg.Args[1]
			var stream io.Writer
			if tag == "stdout" {
				stream = os.Stdout
			} else if tag == "stderr" {
				stream = os.Stderr
			} else {
				return fmt.Errorf("unrecognised tag: %s", tag)
			}
			fmt.Fprint(stream, chunk)
		}
		return nil
	}
	if args[0] == "inspect" {
		if len(args) != 2 {
			return fmt.Errorf("usage: inspect CONTAINER")
		}
		_, container, err := instance.Attach(args[1])
		if err != nil {
			return fmt.Errorf("attach: %v", err)
		}
		json, err := container.Get()
		if err != nil {
			return fmt.Errorf("get: %v", err)
		}
		fmt.Println(json)
		return nil
	}
	return fmt.Errorf("unrecognised command: %s", args[0])
}

func Fatalf(msg string, args ...interface{}) {
	if !strings.HasSuffix(msg, "\n") {
		msg = msg + "\n"
	}
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}
