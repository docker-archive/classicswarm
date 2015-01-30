package scheduler

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/filter"
	"github.com/docker/swarm/scheduler/builtin"
	"github.com/docker/swarm/strategy"
	"github.com/samalba/dockerclient"
)

type Scheduler interface {
	Initialize(cluster *cluster.Cluster, strategy strategy.PlacementStrategy, filters []filter.Filter)
	CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error)
	RemoveContainer(container *cluster.Container, force bool) error
}

var (
	schedulers      map[string]Scheduler
	ErrNotSupported = errors.New("scheduler not supported")
)

func init() {
	schedulers = map[string]Scheduler{
		"builtin": &builtin.BuiltinScheduler{},
	}
}

func New(name string, cluster *cluster.Cluster, strategy strategy.PlacementStrategy, filters []filter.Filter) (Scheduler, error) {
	if scheduler, exists := schedulers[name]; exists {
		log.WithField("name", name).Debug("Initializing scheduler")
		scheduler.Initialize(cluster, strategy, filters)
		return scheduler, nil
	}
	return nil, ErrNotSupported
}
