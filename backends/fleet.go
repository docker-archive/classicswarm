package backends

import (
	fleetJob "github.com/coreos/fleet/job"
	"github.com/coreos/fleet/registry"
	"github.com/coreos/fleet/third_party/github.com/coreos/go-etcd/etcd"
	"github.com/coreos/fleet/unit"
	"github.com/dotcloud/docker/engine"
	"github.com/flynn/go-shlex"

	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func Fleet() engine.Installer {
	return &fleet{}
}

type fleet struct{}

func (f *fleet) Install(eng *engine.Engine) error {
	// TODO: better way to seed the PRNG
	rand.Seed(time.Now().UnixNano())

	eng.Register("fleet", func(job *engine.Job) engine.Status {
		if len(job.Args) != 1 {
			return job.Errorf("usage: %s ETCD_URL", job.Name)
		}
		client := newFleetClient(job.Args[0])
		eng.Register("containers", client.containers)
		eng.Register("create", client.create)
		eng.Register("start", client.start)
		eng.Register("container_delete", client.containerDelete)
		return engine.StatusOK
	})
	return nil
}

type fleetClient struct {
	registry registry.Registry
}

func newFleetClient(etcdURL string) *fleetClient {
	etcdClient := etcd.NewClient([]string{etcdURL})
	registry := registry.New(etcdClient, registry.DefaultKeyPrefix)
	return &fleetClient{registry}
}

func (c *fleetClient) containers(job *engine.Job) engine.Status {
	jobs, err := c.registry.GetAllJobs()
	if err != nil {
		return job.Error(err)
	}

	outs := engine.NewTable("Created", 0)
	for _, fj := range jobs {
		out := new(engine.Env)
		out.Set("Id", fj.Name)

		execStart := fj.Unit.Contents["Service"]["ExecStart"][0]
		fullCommand, err := shlex.Split(execStart)
		if err != nil {
			fullCommand = strings.Split(execStart, " ")
		}
		out.Set("Image", fullCommand[2])
		out.Set("Command", strings.Join(fullCommand[3:], " "))

		if *(fj.State) == fleetJob.JobStateLaunched {
			out.Set("Status", "Up 10 seconds")
		} else if *(fj.State) == fleetJob.JobStateInactive {
			out.Set("Status", "Exited (0) 10 seconds ago")
		}
		outs.Add(out)
	}
	outs.WriteListTo(job.Stdout)

	return engine.StatusOK
}

func (c *fleetClient) create(job *engine.Job) engine.Status {
	id := randomHexString(64)
	command := []string{"/usr/bin/docker", "run"}
	command = append(command, job.Getenv("Image"))

	for _, component := range job.GetenvList("Cmd") {
		command = append(command, strconv.Quote(component))
	}

	u := new(unit.Unit)
	u.Contents = map[string]map[string][]string{}
	u.Contents["Unit"] = map[string][]string{}
	u.Contents["Unit"]["Description"] = []string{id}
	u.Contents["Service"] = map[string][]string{}
	u.Contents["Service"]["ExecStart"] = []string{strings.Join(command, " ")}

	flj := fleetJob.NewJob(fmt.Sprintf("%s.service", id), *u)

	err := c.registry.CreateJob(flj)
	if err != nil {
		return job.Error(err)
	}

	job.Stdout.Write([]byte(id))
	return engine.StatusOK
}

func (c *fleetClient) start(job *engine.Job) engine.Status {
	name := fmt.Sprintf("%s.service", job.Args[0])
	j, err := c.registry.GetJob(name)

	if err != nil {
		return job.Errorf("error retrieving Job(%s) from Registry: %v", name, err)
	} else if j == nil {
		return job.Errorf("unable to find Job(%s)", name)
	} else if j.State == nil {
		return job.Errorf("unable to determine current state of Job")
	} else if *(j.State) == fleetJob.JobStateLaunched {
		job.Logf("Job(%s) already %s, skipping.", j.Name, *(j.State))
		return engine.StatusOK
	}

	job.Logf("Setting Job(%s) target state to launched", j.Name)
	err = c.registry.SetJobTargetState(j.Name, fleetJob.JobStateLaunched)
	if err != nil {
		return job.Error(err)
	}

	return engine.StatusOK
}

func (c *fleetClient) containerDelete(job *engine.Job) engine.Status {
	name := job.Args[0]
	j, err := c.getJobByPartialId(name)
	if err != nil {
		return job.Error(err)
	}
	if j == nil {
		return job.Errorf("unable to find Job(%s)", name)
	}

	err = c.registry.DestroyJob(j.Name)
	if err != nil {
		return job.Error(err)
	}

	return engine.StatusOK
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
