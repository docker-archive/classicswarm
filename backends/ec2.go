package backends

import (
  "time"
  "fmt"
  "strings"
  "github.com/docker/libswarm/beam"

  "launchpad.net/goamz/aws"
  "launchpad.net/goamz/ec2"
)

type ec2Config struct {
  securityGroup string
  instanceType  string
  keypair        string
  zone           string
  ami            string
  tag            string
  region         aws.Region
}

type ec2Client struct {
  config  *ec2Config
  ec2Conn *ec2.EC2
  Server  *beam.Server
  instance *ec2.Instance
}

func (c *ec2Client) get(ctx *beam.Message) error {
  fmt.Println("*** ec2 Get ***")
  body := `{}`
  ctx.Ret.Send(&beam.Message{Verb: beam.Set, Args: []string{body}})
  return nil
}

func (c *ec2Client) start(ctx *beam.Message) error {
  fmt.Println("*** ec2 OnStart ***")
  if err := c.startInstance(); err != nil {
    return err
  }

  if err := c.tagtInstance(); err != nil {
    return err
  }

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

func defaultConfigValues() (config *ec2Config) {
  config               = new(ec2Config)
  config.region        = aws.USEast
  config.instanceType  = "t1.micro"
  return config
}

func newConfig(args []string) (config *ec2Config, err error) {
  // TODO (aaron): fail fast on incorrect number of args

  var optValPair []string
  var opt, val string

  config = defaultConfigValues()

  for _, value := range args {
    optValPair = strings.Split(value, "=")
    opt, val   = optValPair[0], optValPair[1]

    switch opt {
      case "--region":
        config.region = convertToRegion(val)
      case "--zone":
        config.zone = val
      case "--tag":
        config.tag = val
      case "--ami":
        config.ami = val
      case "--keypair":
        config.keypair = val
      case "--security_group":
        config.securityGroup = val
      case "--instance_type":
        config.instanceType = val
      default:
        fmt.Printf("Unrecognizable option: %s value: %s", opt, val)
    }
  }
  return config, nil
}

func convertToRegion(input string) aws.Region {
  // TODO (aaron): fill in more regions
  switch input {
    case "us-east-1":
      return  aws.USEast
    case "us-west-1":
      return  aws.USWest
    case "us-west-2":
      return  aws.USWest2
    case "eu-west-1":
      return  aws.EUWest
    default:
      fmt.Println("Unrecognizable region, default to: us-east-1")
      return aws.USEast
  }
}

func awsInit(config *ec2Config)  (ec2Conn *ec2.EC2, err error) {
  auth, err := aws.EnvAuth()

  if err != nil {
    return nil, err
  }

  return ec2.New(auth, config.region), nil
}

func (c *ec2Client) tagtInstance() error {
  ec2Tags := []ec2.Tag{ec2.Tag{"Name", c.config.tag}}
  if _, err := c.ec2Conn.CreateTags([]string{c.instance.InstanceId}, ec2Tags); err != nil {
    return err
  }
  return nil
}

func (c *ec2Client) startInstance() error {
  options := ec2.RunInstances{
    ImageId:      c.config.ami,
    InstanceType: c.config.instanceType,
    KeyName:      c.config.keypair,
  }

  resp, err := c.ec2Conn.RunInstances(&options)
  if err != nil {
    return err
  }

  // TODO (aaron): this really could be multiple instances, not just 1
  i := resp.Instances[0]

  for i.State.Name != "running" {
    time.Sleep(3 * time.Second)
    fmt.Printf("Waiting for instance to come up.  Current State: %s\n",
               i.State.Name)

    resp, err := c.ec2Conn.Instances([]string{i.InstanceId}, ec2.NewFilter())

    if err != nil {
      return err
    }

    i = resp.Reservations[0].Instances[0]
  }

  c.instance = &i

  fmt.Printf("Instance up and running - id: %s\n", i.InstanceId)
  return nil
}

func Ec2() beam.Sender {
  backend := beam.NewServer()
  backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
    fmt.Println("*** init ***")

    var config, err = newConfig(ctx.Args)

    if (err != nil) {
      return err
    }

    ec2Conn, err := awsInit(config)
    if (err != nil) {
      return err
    }

    client := &ec2Client{config, ec2Conn, beam.NewServer(), nil}
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
