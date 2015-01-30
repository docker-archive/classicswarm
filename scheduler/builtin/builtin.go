package builtin

import (
	"sync"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/filter"
	"github.com/docker/swarm/strategy"
	"github.com/samalba/dockerclient"
)

type BuiltinScheduler struct {
	sync.Mutex

	cluster  *cluster.Cluster
	strategy strategy.PlacementStrategy
	filters  []filter.Filter
}

func (s *BuiltinScheduler) Initialize(cluster *cluster.Cluster, strategy strategy.PlacementStrategy, filters []filter.Filter) {
	s.cluster = cluster
	s.strategy = strategy
	s.filters = filters
}

// Find a nice home for our container.
func (s *BuiltinScheduler) selectNodeForContainer(config *dockerclient.ContainerConfig) (*cluster.Node, error) {
	candidates := s.cluster.Nodes()

	accepted, err := filter.ApplyFilters(s.filters, config, candidates)
	if err != nil {
		return nil, err
	}

	return s.strategy.PlaceContainer(config, accepted)
}

// Schedule a brand new container into the cluster.
func (s *BuiltinScheduler) CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error) {
	/*Disable for now
	if config.Memory == 0 || config.CpuShares == 0 {
		return nil, fmt.Errorf("Creating containers in clustering mode requires resource constraints (-c and -m) to be set")
	}
	*/

	s.Lock()
	defer s.Unlock()

	node, err := s.selectNodeForContainer(config)
	if err != nil {
		return nil, err
	}
	return s.cluster.DeployContainer(node, config, name)
}

// Remove a container from the cluster. Containers should always be destroyed
// through the scheduler to guarantee atomicity.
func (s *BuiltinScheduler) RemoveContainer(container *cluster.Container, force bool) error {
	s.Lock()
	defer s.Unlock()

	return s.cluster.DestroyContainer(container, force)
}
