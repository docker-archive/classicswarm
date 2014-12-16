package scheduler

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

type Scheduler interface {
	Initialize(cluster *cluster.Cluster, c *cli.Context) error
	// Given a container configuration and name, create and return a new container
	CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error)
	RemoveContainer(container *cluster.Container, force bool) error
}

var (
	schedulers      map[string]Scheduler
	ErrNotSupported = errors.New("scheduler not supported")
)

func init() {
	schedulers = map[string]Scheduler{
		"swarm": &SwarmScheduler{},
	}
}

func New(cluster *cluster.Cluster, name string, c *cli.Context) (Scheduler, error) {
	if scheduler, exists := schedulers[name]; exists {
		log.Debugf("Initialising %q scheduler", name)
		err := scheduler.Initialize(cluster, c)
		return scheduler, err
	}
	return nil, ErrNotSupported
}
