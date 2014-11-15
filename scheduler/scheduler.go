package scheduler

import (
	"fmt"
	"sync"

	"github.com/docker/libcluster"
	"github.com/docker/libcluster/scheduler/filter"
	"github.com/docker/libcluster/scheduler/strategy"
	"github.com/samalba/dockerclient"
)

type Scheduler struct {
	sync.Mutex

	cluster  *libcluster.Cluster
	strategy strategy.PlacementStrategy
	filters  []filter.Filter
}

func NewScheduler(cluster *libcluster.Cluster) *Scheduler {
	return &Scheduler{
		cluster:  cluster,
		strategy: &strategy.RandomPlacementStrategy{},
		filters:  []filter.Filter{},
	}
}

// Find a nice home for our container.
func (s *Scheduler) selectNodeForContainer(config *dockerclient.ContainerConfig) (*libcluster.Node, error) {
	candidates := []*libcluster.Node{}
	for _, node := range s.cluster.Nodes() {
		candidates = append(candidates, node)
	}

	accepted, err := filter.ApplyFilters(s.filters, config, candidates)
	if err != nil {
		return nil, err
	}

	return s.strategy.PlaceContainer(config, accepted)
}

// Schedule a brand new container into the cluster.
func (s *Scheduler) CreateContainer(config *dockerclient.ContainerConfig, name string) (*libcluster.Container, error) {
	s.Lock()
	defer s.Unlock()

	if config.Memory == 0 || config.CpuShares == 0 {
		return nil, fmt.Errorf("Creating containers in clustering mode requires resource constraints (-c and -m) to be set")
	}

	node, err := s.selectNodeForContainer(config)
	if err != nil {
		return nil, err
	}
	return node.Create(config, name, true)
}

// Remove a container from the cluster. Containers should always be destroyed
// through the scheduler to guarantee atomicity.
func (s *Scheduler) RemoveContainer(container *libcluster.Container, force bool) error {
	s.Lock()
	defer s.Unlock()

	return container.Node().Remove(container, force)
}
