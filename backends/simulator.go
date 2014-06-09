package backends

import (
	"github.com/docker/libswarm/beam"
	"time"
)

func Simulator() beam.Sender {
	s := beam.NewServer()
	s.OnVerb(beam.Spawn, beam.Handler(func(ctx *beam.Message) error {
		sim := &simulator{ctx.Args, &beam.Object{ctx.Ret}, beam.NewServer()}
		sim.Server.OnVerb(beam.Attach, beam.Handler(sim.attach))
		sim.Server.OnVerb(beam.Start, beam.Handler(sim.start))
		sim.Server.OnVerb(beam.Ls, beam.Handler(sim.ls))
		sim.Server.OnVerb(beam.Spawn, beam.Handler(sim.spawn))
		_, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: sim.Server})
		return err
	}))
	return s
}

type simulator struct {
	containers []string
	out beam.Sender
	*beam.Server
}

func (s *simulator) attach(msg *beam.Message) error {
	if msg.Args[0] == "" {
		msg.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: s.Server})
		for {
			time.Sleep(1 * time.Second)
			msg.Ret.Send(&beam.Message{Verb: beam.Log, Args: []string{"stdout", "heartbeat"}})
		}
	} else {
		c := newContSim(msg.Args[0])
		msg.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c})
	}
	return nil
}

func (s *simulator) start(ctx *beam.Message) error {
	ctx.Ret.Send(&beam.Message{Verb: beam.Ack})
	return nil
}

func (s *simulator) ls(msg *beam.Message) error {
	_, err := msg.Ret.Send(&beam.Message{Verb: beam.Set, Args: s.containers})
	return err
}

func (s *simulator) spawn(msg *beam.Message) error {
	c := newContSim(msg.Args[0])
	msg.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c})
	return nil
}

type contsim struct {
	name string
}

func newContSim(name string) beam.Sender {
	c := &contsim{name}
	instance := beam.NewServer()
	instance.OnVerb(beam.Attach, beam.Handler(c.attach))
	instance.OnVerb(beam.Get, beam.Handler(c.get))
	return instance
}

func (c *contsim) attach(msg *beam.Message) error {
	msg.Ret.Send(&beam.Message{Verb: beam.Ack})
	return nil
}

func (c *contsim) get(msg *beam.Message) error {
	msg.Ret.Send(&beam.Message{Verb: beam.Set, Args: []string{body}})
	return nil
}

const body = `
{"ID":"ce936fa4d6078d82afd2fccc13ae5bf076c495865d70d3666726edfe28fc35a0","Created":"2014-06-08T16:11:58.514764968Z","Path":"/bin/sleep","Args":["300"],"Config":{"Hostname":"ce936fa4d607","Domainname":"","User":"","Memory":0,"MemorySwap":0,"CpuShares":0,"AttachStdin":false,"AttachStdout":true,"AttachStderr":true,"PortSpecs":null,"ExposedPorts":null,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":["HOME=/","PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sleep","300"],"Image":"busybox","Volumes":null,"WorkingDir":"","Entrypoint":null,"NetworkDisabled":false,"OnBuild":null},"State":{"Running":true,"Pid":7226,"ExitCode":0,"StartedAt":"2014-06-08T16:11:58.669325638Z","FinishedAt":"0001-01-01T00:00:00Z"},"Image":"a9eb172552348a9a49180694790b33a1097f546456d041b6e82e4d7716ddb721","NetworkSettings":{"IPAddress":"10.0.0.2","IPPrefixLen":16,"Gateway":"10.0.42.1","Bridge":"docker0","PortMapping":null,"Ports":{}},"ResolvConfPath":"/etc/resolv.conf","HostnamePath":"/var/lib/docker/containers/ce936fa4d6078d82afd2fccc13ae5bf076c495865d70d3666726edfe28fc35a0/hostname","HostsPath":"/var/lib/docker/containers/ce936fa4d6078d82afd2fccc13ae5bf076c495865d70d3666726edfe28fc35a0/hosts","Name":"/trusting_feynman","Driver":"btrfs","ExecDriver":"native-0.2","MountLabel":"","ProcessLabel":"","Volumes":{},"VolumesRW":{},"HostConfig":{"Binds":null,"ContainerIDFile":"","LxcConf":[],"Privileged":false,"PortBindings":{},"Links":null,"PublishAllPorts":false,"Dns":null,"DnsSearch":null,"VolumesFrom":null,"NetworkMode":"bridge"}}
`
