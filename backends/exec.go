package backends

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/docker/libswarm"
)

func Exec() libswarm.Sender {
	e := libswarm.NewServer()
	e.OnVerb(libswarm.Spawn, libswarm.Handler(func(msg *libswarm.Message) error {
		if len(msg.Args) < 1 {
			return fmt.Errorf("usage: SPAWN exec|... <config>")
		}
		if msg.Args[0] != "exec" {
			return fmt.Errorf("invalid command: %s", msg.Args[0])
		}
		var config struct {
			Path string
			Args []string
		}
		if err := json.Unmarshal([]byte(msg.Args[1]), &config); err != nil {
			config.Path = msg.Args[1]
			config.Args = msg.Args[2:]
		}
		cmd := &command{
			Cmd:    exec.Command(config.Path, config.Args...),
			Server: libswarm.NewServer(),
		}
		cmd.OnVerb(libswarm.Attach, libswarm.Handler(func(msg *libswarm.Message) error {
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				return err
			}
			stdin, err := cmd.StdinPipe()
			if err != nil {
				return err
			}
			inR, inW := libswarm.Pipe()
			if _, err := msg.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: inW}); err != nil {
				return err
			}
			out := libswarm.AsClient(msg.Ret)
			go func() {
				defer stdin.Close()
				for {
					msg, err := inR.Receive(0)
					if err != nil {
						return
					}
					if msg.Verb == libswarm.Log && len(msg.Args) > 0 {
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
		cmd.OnVerb(libswarm.Start, libswarm.Handler(func(msg *libswarm.Message) error {
			cmd.tasks.Add(1)
			if err := cmd.Cmd.Start(); err != nil {
				return err
			}
			go func() {
				defer cmd.tasks.Done()
				if err := cmd.Cmd.Wait(); err != nil {
					libswarm.AsClient(msg.Ret).Log("%s exited status=%v", cmd.Cmd.Path, err)
				}
			}()
			msg.Ret.Send(&libswarm.Message{Verb: libswarm.Ack})
			return nil
		}))
		if _, err := msg.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: cmd}); err != nil {
			return err
		}
		return nil
	}))
	return e
}

type command struct {
	*exec.Cmd
	*libswarm.Server
	tasks sync.WaitGroup
}
