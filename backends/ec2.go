package backends

import (
  "time"
  "fmt"
  "strings"
  "github.com/docker/libswarm/beam"
)

type ec2Config struct {
  access_key     string
  secret_key     string
  security_group string
  instance_type  string
  keypair        string
  region         string
  zone           string
  ami            string
  tag            string
}

type ec2Client struct {
  config *ec2Config
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

  for {
     time.Sleep(1 * time.Second)
     (&beam.Object{ctx.Ret}).Log("ec2: heartbeat")
  }
  return nil
}

func newConfig(args []string) (config *ec2Config, err error) {
  // TODO (aaron): fail fast on incorrect number of args

  var optValPair []string
  var opt, val string

  config = new(ec2Config)
  for _, value := range args {
    optValPair = strings.Split(value, "=")
    opt = optValPair[0]
    val = optValPair[1]

    switch opt {
      case "--region":
        config.region = val
      case "--zone":
        config.zone = val
      case "--tag":
        config.tag = val
      case "--ami":
        config.ami = val
      case "--keypair":
        config.keypair = val
      case "--security_group":
        config.security_group = val
      case "--instance_type":
        config.instance_type = val
      default:
        fmt.Printf("Unrecognizable option: %s value: %s", opt, val)
    }
  }
  return config, nil
}

func Ec2() beam.Sender {
  backend := beam.NewServer()
  backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
    fmt.Println("*** init ***")

    var config, err = newConfig(ctx.Args)

    if (err != nil) {
      return err
    }

    client := &ec2Client{config, beam.NewServer()}
    client.Server.OnSpawn(beam.Handler(client.spawn))
    client.Server.OnStart(beam.Handler(client.start))
    client.Server.OnStop(beam.Handler(client.stop))
    client.Server.OnAttach(beam.Handler(client.attach))
    client.Server.OnLog(beam.Handler(client.log))
    client.Server.OnError(beam.Handler(client.error))
    client.Server.OnLs(beam.Handler(client.ls))
    client.Server.OnGet(beam.Handler(client.get))
    _, err = ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: client.Server})
    return err
  }))
  return backend
}
