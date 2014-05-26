package backends

import (
	fleetJob "github.com/coreos/fleet/job"
	"github.com/coreos/fleet/registry"
	"github.com/coreos/fleet/third_party/github.com/coreos/go-etcd/etcd"
	"github.com/coreos/fleet/unit"
	"github.com/dotcloud/docker/engine"

	"encoding/json"
	"fmt"
	"math/rand"
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
	serialized, err := json.Marshal(jobs)
	if err != nil {
		return job.Error(err)
	}
	job.Stdout.Write(serialized)
	return engine.StatusOK
}

func (c *fleetClient) create(job *engine.Job) engine.Status {
	id := randomHexString(64)

	u := unit.NewUnit(fmt.Sprintf(`[Unit]
Description=%s

[Service]
ExecStart=/bin/bash -c "while true; do echo \"Hello, world\"; sleep 1; done"
`, id))

	flj := fleetJob.NewJob(id, *u)

	err := c.registry.CreateJob(flj)
	if err != nil {
		return job.Error(err)
	}

	job.Stdout.Write([]byte(id))
	return engine.StatusOK
}

func (c *fleetClient) start(job *engine.Job) engine.Status {
	name := job.Args[0]
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
	c.registry.SetJobTargetState(j.Name, fleetJob.JobStateLaunched)

	return engine.StatusOK
}

func randomHexString(n int) string {
	b := make([]byte, n/2+1)
	for i := 0; i < len(b); i++ {
		b[i] = byte(rand.Intn(0x100))
	}
	return fmt.Sprintf("%x", b)[0:n]
}
