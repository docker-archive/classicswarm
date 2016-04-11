package scheduler

import (
	"errors"
	"strings"
	"sync"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/filter"
	"github.com/docker/swarm/scheduler/node"
	"github.com/docker/swarm/scheduler/strategy"
)

var (
	errNoNodeAvailable = errors.New("No nodes available in the cluster")
)

// Scheduler is exported
type Scheduler struct {
	sync.Mutex

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

// SelectNodesForContainer will return a list of nodes where the container can
// be scheduled, sorted by order or preference.
func (s *Scheduler) SelectNodesForContainer(nodes []*node.Node, config *cluster.ContainerConfig) ([]*node.Node, error) {
	candidates, err := s.selectNodesForContainer(nodes, config, true)

	if err != nil {
		candidates, err = s.selectNodesForContainer(nodes, config, false)
	}
	return candidates, err
}

func (s *Scheduler) selectNodesForContainer(nodes []*node.Node, config *cluster.ContainerConfig, soft bool) ([]*node.Node, error) {
	nominees, err := filter.ApplyFilters(s.filters, config, nodes, soft)
	if err != nil {
		return nil, err
	}

	if len(nominees) == 0 {
		return nil, errNoNodeAvailable
	}

	accepted := make([]*node.Node, 0)
	// TODO: potentially move filtering of hosts in maintenance mode into Applyfilters or another helper
	for _, n := range nominees {
		if n.MaintenanceMode == false {
			accepted = append(accepted, n)
		}
	}

	if len(accepted) == 0 {
		return nil, errNoNodeAvailable
	}

	return s.strategy.RankAndSort(config, accepted)
}

// Strategy returns the strategy name
func (s *Scheduler) Strategy() string {
	return s.strategy.Name()
}

// Filters returns the list of filter's name
func (s *Scheduler) Filters() string {
	filters := []string{}
	for _, f := range s.filters {
		filters = append(filters, f.Name())
	}

	return strings.Join(filters, ", ")
}
