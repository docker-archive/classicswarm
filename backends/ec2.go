package backends

import (
  "fmt"
  "github.com/docker/libswarm/beam"
)

type ec2Client struct {
  Server *beam.Server
}

func (c *ec2Client) get(ctx *beam.Message) error {
  fmt.Println("*** ec2 Get ***")
  body := `{}`
  ctx.Ret.Send(&beam.Message{Verb: beam.Set, Args: []string{body}})
  return nil
}

func (c *ec2Client) start(ctx *beam.Message) error {
  fmt.Println("*** ec2 OnStart ***")
  ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c.Server})
  return nil
}

func (c *ec2Client) log(ctx *beam.Message) error {
  fmt.Println("*** ec2 OnLog ***")
  ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c.Server})
  return nil
}

func (c *ec2Client) spawn(ctx *beam.Message) error {
  fmt.Println("*** ec2 OnSpawn ***")
  ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c.Server})
  return nil
}

func (c *ec2Client) ls(ctx *beam.Message) error {
  fmt.Println("*** ec2 OnLs ***")
  ctx.Ret.Send(&beam.Message{Verb: beam.Set, Args: []string{}})
  return nil
}

func (c *ec2Client) error(ctx *beam.Message) error {
  fmt.Println("*** ec2 OnError ***")
  ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c.Server})
  return nil
}

func (c *ec2Client) stop(ctx *beam.Message) error {
  fmt.Println("*** ec2 OnStop ***")
  ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c.Server})
  return nil
}

func (c *ec2Client) attach(ctx *beam.Message) error {
  fmt.Println("*** ec2 OnAttach ***")
  ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c.Server})
  return nil
}

func Ec2() beam.Sender {
  backend := beam.NewServer()
  backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
    fmt.Println("*** init ***")
    client := &ec2Client{beam.NewServer()}
    client.Server.OnSpawn(beam.Handler(client.spawn))
    client.Server.OnStart(beam.Handler(client.start))
    client.Server.OnStop(beam.Handler(client.stop))
    client.Server.OnAttach(beam.Handler(client.attach))
    client.Server.OnLog(beam.Handler(client.log))
    client.Server.OnError(beam.Handler(client.error))
    client.Server.OnLs(beam.Handler(client.ls))
    client.Server.OnGet(beam.Handler(client.get))
    _, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: client.Server})
    return err
  }))
  return backend
}
