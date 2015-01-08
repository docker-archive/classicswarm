package scheduler

import (
	"sync"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/filter"
	"github.com/docker/swarm/scheduler/strategy"
	"github.com/samalba/dockerclient"
)

type Scheduler struct {
	sync.Mutex

	cluster  *cluster.Cluster
	strategy strategy.PlacementStrategy
	filters  []filter.Filter
}

func NewScheduler(cluster *cluster.Cluster, strategy strategy.PlacementStrategy, filters []filter.Filter) *Scheduler {
	return &Scheduler{
		cluster:  cluster,
		strategy: strategy,
		filters:  filters,
	}
}

// Find a nice home for our container.
func (s *Scheduler) SelectNodeForContainer(config *dockerclient.ContainerConfig) (*cluster.Node, error) {
	s.Lock()
	defer s.Unlock()

	candidates := s.cluster.Nodes()

	accepted, err := filter.ApplyFilters(s.filters, config, candidates)
	if err != nil {
		return nil, err
	}

	return s.strategy.PlaceContainer(config, accepted)
}

// Schedule a brand new container into the cluster.
func (s *Scheduler) CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error) {
	/*Disable for now
	if config.Memory == 0 || config.CpuShares == 0 {
		return nil, fmt.Errorf("Creating containers in clustering mode requires resource constraints (-c and -m) to be set")
	}
	*/

	node, err := s.SelectNodeForContainer(config)
	if err != nil {
		return nil, err
	}
	return node.Create(config, name, true)
}

// Remove a container from the cluster. Containers should always be destroyed
// through the scheduler to guarantee atomicity.
func (s *Scheduler) RemoveContainer(container *cluster.Container, force bool) error {
	s.Lock()
	defer s.Unlock()

	return container.Node().Destroy(container, force)
}
