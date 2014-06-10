package backends

import (
	"encoding/json"
	"github.com/docker/libswarm/beam"
	"strings"
	"time"

	fleetJob "github.com/coreos/fleet/job"
	"github.com/coreos/fleet/registry"
	"github.com/coreos/fleet/third_party/github.com/coreos/go-etcd/etcd"
	"github.com/coreos/fleet/unit"
	"github.com/samalba/dockerclient"

	"fmt"
	"math/rand"
	"strconv"
)

func FleetDocker() beam.Sender {
	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(func(msg *beam.Message) error {
		url := "http://localhost:4001"
		if len(msg.Args) > 0 {
			url = msg.Args[0]
		}
		etcdClient := etcd.NewClient([]string{url})
		registry := registry.New(etcdClient, registry.DefaultKeyPrefix)
		f := &fleetClient{registry, beam.NewServer()}

		f.Server.OnAttach(beam.Handler(f.attach))
		f.Server.OnStart(beam.Handler(f.start))
		f.Server.OnLs(beam.Handler(f.ls))
		f.Server.OnSpawn(beam.Handler(f.spawn))

		_, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: f.Server})
		return err
	}))
	return backend
}

type fleetClient struct {
	registry registry.Registry
	*beam.Server
}

func (c *fleetClient) newUnit(unit string) beam.Sender {
	u := &fleetUnit{unit, c}
	instance := beam.NewServer()
	instance.OnAttach(beam.Handler(u.attach))
	instance.OnStart(beam.Handler(u.start))
	instance.OnStop(beam.Handler(u.stop))
	instance.OnGet(beam.Handler(u.get))
	return instance
}

func (c *fleetClient) attach(msg *beam.Message) error {
	if msg.Args[0] == "" {
		msg.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c.Server})
		for {
			time.Sleep(1 * time.Second)
			(&beam.Object{msg.Ret}).Log("fleet: heartbeat")
		}
	} else {
		u := c.newUnit(msg.Args[0])
		msg.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: u})
	}

	return nil
}

func (c *fleetClient) ls(msg *beam.Message) error {
	jobs, err := c.registry.GetAllJobs()
	if err != nil {
		return err
	}

	names := []string{}
	for _, fj := range jobs {
		if strings.HasPrefix(fj.Name, "docker-") {
			names = append(names, fj.Name)
		}
	}

	if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Set, Args: names}); err != nil {
		return fmt.Errorf("send response: %v", err)
	}

	return nil
}

func (c *fleetClient) spawn(msg *beam.Message) error {
	id := randomHexString(16)

	config := &dockerclient.ContainerConfig{}
	err := json.Unmarshal([]byte(msg.Args[0]), config)
	if err != nil {
		return err
	}

	rm := []string{"/usr/bin/docker", "rm", id}
	command := []string{"/usr/bin/docker", "run"}
	command = append(command, config.Image)

	for _, component := range config.Cmd {
		command = append(command, strconv.Quote(component))
	}

	// hack(philips): use a real deserializer
	template := `
[Unit]
Description=%s
[Service]
ExecStartPre=-%s
ExecStart=%s
[X-Docker]
Image=%s
Cmd=%s
`
	u := unit.NewUnit(fmt.Sprintf(template, id, strings.Join(rm, " "), strings.Join(command, " "), config.Image, config.Cmd))

	name := fmt.Sprintf("docker-%s.service", id)
	flj := fleetJob.NewJob(name, *u)

	err = c.registry.CreateJob(flj)
	if err != nil {
		return err
	}

	err = c.registry.SetJobTargetState(name, fleetJob.JobStateLaunched)
	if err != nil {
		return err
	}

	return nil
}

func (f *fleetClient) start(msg *beam.Message) error {
	msg.Ret.Send(&beam.Message{Verb: beam.Ack})
	return nil
}

func (c *fleetClient) getJobByPartialId(name string) (*fleetJob.Job, error) {
	jobs, err := c.registry.GetAllJobs()
	if err != nil {
		return nil, err
	}

	for _, j := range jobs {
		if j.Name[0:len(name)] == name {
			return &j, nil
		}
	}

	return nil, nil
}

func randomHexString(n int) string {
	b := make([]byte, n/2+1)
	for i := 0; i < len(b); i++ {
		b[i] = byte(rand.Intn(0x100))
	}
	return fmt.Sprintf("%x", b)[0:n]
}

type fleetUnit struct {
	unit string
	client *fleetClient
}

func (u *fleetUnit) attach(msg *beam.Message) error {
	_, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack})
	return err
}

func (u *fleetUnit) start(msg *beam.Message) error {
	name := fmt.Sprintf("docker-%s.service", u.unit)
	j, err := u.client.registry.GetJob(name)

	if err != nil {
		return fmt.Errorf("error retrieving Job(%s) from Registry: %v", name, err)
	} else if j == nil {
		return fmt.Errorf("unable to find Job(%s)", name)
	} else if j.State == nil {
		return fmt.Errorf("unable to determine current state of Job")
	} else if *(j.State) == fleetJob.JobStateLaunched {
		(&beam.Object{msg.Ret}).Log(fmt.Sprintf("Job(%s) already %s, skipping.", j.Name, *(j.State)))
		return nil
	}

	(&beam.Object{msg.Ret}).Log(fmt.Sprintf("Setting Job(%s) target state to launched", j.Name))
	err = u.client.registry.SetJobTargetState(j.Name, fleetJob.JobStateLaunched)
	if err != nil {
		return err
	}

	msg.Ret.Send(&beam.Message{Verb: beam.Ack})

	return nil
}

func (u *fleetUnit) stop(msg *beam.Message) error {
	j, err := u.client.getJobByPartialId(u.unit)
	if err != nil {
		return err
	}
	if j == nil {
		return fmt.Errorf("unable to find Job(%s)", u.unit)
	}

	err = u.client.registry.DestroyJob(j.Name)
	if err != nil {
		return err
	}

	msg.Ret.Send(&beam.Message{Verb: beam.Ack})

	return nil
}

func (u *fleetUnit) get(msg *beam.Message) error {
	i := dockerclient.ContainerInfo{}
	j, err := u.client.getJobByPartialId(u.unit)

	i.Id = u.unit
	i.Name = "/"+u.unit
	i.Path = u.unit
	i.Args = []string{}
	i.Created = time.Now().Format(time.RFC3339)
	i.State.StartedAt = time.Now().Format(time.RFC3339)
	i.State.Running = (j.UnitState.ActiveState == "active")
	i.Config = &dockerclient.ContainerConfig{
		Cmd: j.Unit.Contents["X-Docker"]["Cmd"],
		Image: j.Unit.Contents["X-Docker"]["Image"][0],
	}
	i.HostConfig = &dockerclient.HostConfig{}

	data, err := json.Marshal(i)
	if err != nil {
		return err
	}

	msg.Ret.Send(&beam.Message{Verb: beam.Set, Args: []string{string(data)}})
	return nil
}
