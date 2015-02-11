package builtin

import (
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
	"github.com/docker/swarm/filter"
	"github.com/docker/swarm/scheduler/options"
	"github.com/samalba/dockerclient"
)

type BuiltinScheduler struct {
	sync.Mutex

	cluster *cluster.Cluster
	options *options.SchedulerOptions
}

func (s *BuiltinScheduler) Initialize(options *options.SchedulerOptions) {
	s.options = options

	s.cluster = cluster.NewCluster(s.options.Store)
}

// Schedule a brand new container into the cluster.
func (s *BuiltinScheduler) CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error) {

	s.Lock()
	defer s.Unlock()

	candidates := s.cluster.Nodes()

	// Find a nice home for our container.
	accepted, err := filter.ApplyFilters(s.options.Filters, config, candidates)
	if err != nil {
		return nil, err
	}

	node, err := s.options.Strategy.PlaceContainer(config, accepted)
	if err != nil {
		return nil, err
	}

	container, err := node.Create(config, name, true)
	if err != nil {
		return nil, err
	}

	return container, s.cluster.CommitContainerInStore(container.Id, config, name)
}

// Remove a container from the cluster. Containers should always be destroyed
// through the scheduler to guarantee atomicity.
func (s *BuiltinScheduler) RemoveContainer(container *cluster.Container, force bool) error {
	s.Lock()
	defer s.Unlock()

	if err := container.Node.Destroy(container, force); err != nil {
		return err
	}

	return s.cluster.RemoveContainerFromStore(container.Id, force)
}

// Entries are Docker Nodes
func (s *BuiltinScheduler) NewEntries(entries []*discovery.Entry) {
	for _, entry := range entries {
		go func(m *discovery.Entry) {
			if s.cluster.Node(m.String()) == nil {
				n := cluster.NewNode(m.String(), s.options.OvercommitRatio)
				if err := n.Connect(s.options.TLSConfig); err != nil {
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
