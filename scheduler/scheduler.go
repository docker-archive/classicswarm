package scheduler

import (
	"errors"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

type Scheduler interface {
	Initialize(cluster *cluster.Cluster, opts map[string]string) error
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
		"api":   &ApiScheduler{},
	}
}

func New(cluster *cluster.Cluster, name string, stringOpts cli.StringSlice) (Scheduler, error) {
	if scheduler, exists := schedulers[name]; exists {
		var opts = map[string]string{}
		for _, opt := range stringOpts {
			parts := strings.SplitN(opt, ":", 2)
			opts[parts[0]] = parts[1]
		}
		log.Debugf("Initialising %q scheduler with options %q", name, opts)
		err := scheduler.Initialize(cluster, opts)
		return scheduler, err
	}
	return nil, ErrNotSupported
}
