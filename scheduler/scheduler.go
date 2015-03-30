package scheduler

import (
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/filter"
	"github.com/docker/swarm/scheduler/strategy"
	"github.com/samalba/dockerclient"
)

// Scheduler is exported
type Scheduler struct {
	strategy strategy.PlacementStrategy
	filters  []filter.Filter
}

// New is exported
func New(strategy strategy.PlacementStrategy, filters []filter.Filter) *Scheduler {
	return &Scheduler{
		strategy: strategy,
		filters:  filters,
	}
}

// SelectNodeForContainer will find a nice home for our container.
func (s *Scheduler) SelectNodeForContainer(nodes []cluster.Node, config *dockerclient.ContainerConfig) (cluster.Node, error) {
	accepted, err := filter.ApplyFilters(s.filters, config, nodes)
	if err != nil {
		return nil, err
	}

	return s.strategy.PlaceContainer(config, accepted)
}
