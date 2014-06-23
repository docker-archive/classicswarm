package backends

import (
  "os"
  "os/exec"
  "os/signal"
  "syscall"
  "github.com/docker/libswarm/beam"
)

type sshClient struct {
  server *beam.Server
  args   []string
  tunnel *os.Process
}

func (c *sshClient) attach(ctx *beam.Message) error {
  ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c.server})
	<-make(chan struct{})
  return nil
}

func (c *sshClient) start(ctx *beam.Message) error {
  defaultOptions := []string {
	  "-o", "PasswordAuthentication=no",
	  "-o", "LogLevel=quiet",
	  "-o", "UserKnownHostsFile=/dev/null",
	  "-o", "CheckHostIP=no",
	  "-o", "StrictHostKeyChecking=no",
	  "-A",
		"-N",
	  "-f",
  }

  var allOptions []string
  if len(c.args) > 0 {
    allOptions = append(defaultOptions, c.args...)
  } else {
    allOptions = append(defaultOptions, ctx.Args...)
  }

	cmd := exec.Command("ssh", allOptions...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

  c.tunnel = cmd.Process

  ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c.server})
  return nil
}

func (c *sshClient) Close() {
  if c.tunnel != nil {
    c.tunnel.Kill()
	  c.tunnel.Wait()
		c.tunnel = nil
  }
}

func handleSignal(client *sshClient) {
  c := make(chan os.Signal, 1)
  signal.Notify(c, os.Interrupt)
  signal.Notify(c, syscall.SIGTERM)
  go func() {
      <-c
      client.Close()
      os.Exit(0)
  }()
}

func Ssh() beam.Sender {
  backend := beam.NewServer()
  backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
    client := &sshClient{beam.NewServer(), ctx.Args, nil}

    client.server.OnStart(beam.Handler(client.start))
    client.server.OnAttach(beam.Handler(client.attach))

    handleSignal(client)
    _, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: client.server})

    return err
  }))

  return backend
}

