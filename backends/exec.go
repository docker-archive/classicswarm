package backends

import (
	"fmt"
	"io"
	"os/exec"
	"encoding/json"
	"bufio"
	"strings"
	"sync"

	"github.com/docker/libswarm/beam"
)

func Exec() beam.Sender {
	e := beam.NewServer()
	e.OnSpawn(beam.Handler(func(msg *beam.Message) error {
		if len(msg.Args) < 1 {
			return fmt.Errorf("usage: SPAWN exec|... <config>")
		}
		if msg.Args[0] != "exec" {
			return fmt.Errorf("invalid command: %s", msg.Args[0])
		}
		var config struct {
			Path	string
			Args	[]string
		}
		if err := json.Unmarshal([]byte(msg.Args[1]), &config); err != nil {
			config.Path = msg.Args[1]
			config.Args = msg.Args[2:]
		}
		cmd := &command{
			Cmd:	exec.Command(config.Path, config.Args...),
			Server: beam.NewServer(),
		}
		cmd.OnAttach(beam.Handler(func(msg *beam.Message) error {
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				return err
			}
			stdin, err := cmd.StdinPipe()
			if err != nil {
				return err
			}
			inR, inW := beam.Pipe()
			if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: inW}); err != nil {
				return err
			}
			out := beam.Obj(msg.Ret)
			go func() {
				defer stdin.Close()
				for {
					msg, err := inR.Receive(0)
					if err != nil {
						return
					}
					if msg.Verb == beam.Log && len(msg.Args) > 0 {
						fmt.Fprintf(stdin, "%s\n", strings.TrimRight(msg.Args[0], "\r\n"))
					}
				}
			}()
			cmd.tasks.Add(1)
			go func() {
				defer cmd.tasks.Done()
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					if scanner.Err() != io.EOF && scanner.Err() != nil {
						return
					}
					if err := out.Log(scanner.Text()); err != nil {
						out.Error("%v", err)
						return
					}
				}
			}()
			cmd.tasks.Wait()
			return nil
		}))
		cmd.OnStart(beam.Handler(func(msg *beam.Message) error {
			cmd.tasks.Add(1)
			if err := cmd.Cmd.Start(); err != nil {
				return err
			}
			go func() {
				defer cmd.tasks.Done()
				if err := cmd.Cmd.Wait(); err != nil {
					beam.Obj(msg.Ret).Log("%s exited status=%v", cmd.Cmd.Path, err)
				}
			}()
			msg.Ret.Send(&beam.Message{Verb: beam.Ack})
			return nil
		}))
		if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: cmd}); err != nil {
			return err
		}
		return nil
	}))
	return e
}

type command struct {
	*exec.Cmd
	*beam.Server
	tasks sync.WaitGroup
}
