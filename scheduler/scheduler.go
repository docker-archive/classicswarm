package scheduler

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
	"github.com/docker/swarm/filter"
	"github.com/docker/swarm/scheduler/builtin"
	"github.com/docker/swarm/scheduler/mesos"
	"github.com/docker/swarm/strategy"
	"github.com/samalba/dockerclient"
)

type Scheduler interface {
	Initialize(cluster *cluster.Cluster, strategy strategy.PlacementStrategy, filters []filter.Filter)
	CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error)
	RemoveContainer(container *cluster.Container, force bool) error

	Events(eventsHandler cluster.EventHandler)
	Nodes() []*cluster.Node
	Containers() []*cluster.Container
	Container(IdOrName string) *cluster.Container

	NewEntries(entries []*discovery.Entry)
}

var schedulers map[string]Scheduler

func init() {
	schedulers = map[string]Scheduler{
		"builtin": &builtin.BuiltinScheduler{},
		"mesos":   &mesos.MesosScheduler{},
	}
}

func New(name string, cluster *cluster.Cluster, strategy strategy.PlacementStrategy, filters []filter.Filter) (Scheduler, error) {
	if scheduler, exists := schedulers[name]; exists {
		log.WithField("name", name).Debug("Initializing scheduler")
		scheduler.Initialize(cluster, strategy, filters)
		return scheduler, nil
	}
	return nil, fmt.Errorf("scheduler %q not supported", name)
}
