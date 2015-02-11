package builtin

import (
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
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

// Entries are Docker Nodes
func (s *BuiltinScheduler) NewEntries(entries []*discovery.Entry) {
	for _, entry := range entries {
		go func(m *discovery.Entry) {
			if s.cluster.Node(m.String()) == nil {
				n := cluster.NewNode(m.String(), s.cluster.OvercommitRatio)
				if err := n.Connect(s.cluster.TLSConfig); err != nil {
					log.Error(err)
					return
				}
				if err := s.cluster.AddNode(n); err != nil {
					log.Error(err)
					return
				}
			}
		}(entry)
	}
}

func (s *BuiltinScheduler) Events(eventsHandler cluster.EventHandler) {
	s.cluster.Events(eventsHandler)
}

func (s *BuiltinScheduler) Nodes() []*cluster.Node {
	return s.cluster.Nodes()
}

func (s *BuiltinScheduler) Containers() []*cluster.Container {
	return s.cluster.Containers()
}

func (s *BuiltinScheduler) Container(IdOrName string) *cluster.Container {
	return s.cluster.Container(IdOrName)
}
