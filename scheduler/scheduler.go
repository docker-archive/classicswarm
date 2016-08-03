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

// SelectNodeForContainer will return a "best" node where the container can
// be scheduled.
func (s *Scheduler) SelectNodeForContainer(nodes []*node.Node, config *cluster.ContainerConfig) (*node.Node, error) {
	candidates, err := s.selectNodesForContainer(nodes, config, true)

	if err != nil {
		if candidates, err = s.selectNodesForContainer(nodes, config, false); err != nil {
			return nil, err
		}
	}

	return candidates[0], err
}

func (s *Scheduler) selectNodesForContainer(nodes []*node.Node, config *cluster.ContainerConfig, soft bool) ([]*node.Node, error) {
	accepted, err := filter.ApplyFilters(s.filters, config, nodes, soft)
	if err != nil {
		return nil, err
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
