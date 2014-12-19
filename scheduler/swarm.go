package scheduler

import (
	"strings"
	"sync"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/filter"
	"github.com/docker/swarm/scheduler/strategy"
	"github.com/samalba/dockerclient"
)

type SwarmScheduler struct {
	sync.Mutex

	cluster  *cluster.Cluster
	strategy strategy.PlacementStrategy
	filters  []filter.Filter
}

func (s *SwarmScheduler) Initialize(cluster *cluster.Cluster, opts map[string]string) error {

	// Strategy
	strat, err := strategy.New(opts["strategy"])
	if err != nil {
		return err
	}
	s.strategy = strat

	// Filters
	names := strings.Split(opts["filters"], ",")
	fs, err := filter.New(names)
	if err != nil {
		return err
	}
	s.filters = fs

	s.cluster = cluster
	return nil
}

// Find a nice home for our container.
func (s *SwarmScheduler) selectNodeForContainer(config *dockerclient.ContainerConfig) (*cluster.Node, error) {
	candidates := s.cluster.Nodes()

	accepted, err := filter.ApplyFilters(s.filters, config, candidates)
	if err != nil {
		return nil, err
	}

	return s.strategy.PlaceContainer(config, accepted)
}

// Schedule a brand new container into the cluster.
func (s *SwarmScheduler) CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error) {
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
	return node.Create(config, name, true)
}

// Remove a container from the cluster. Containers should always be destroyed
// through the scheduler to guarantee atomicity.
func (s *SwarmScheduler) RemoveContainer(container *cluster.Container, force bool) error {
	s.Lock()
	defer s.Unlock()

	return container.Node().Destroy(container, force)
}
