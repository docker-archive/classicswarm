package mesos

import (
	"errors"
	"sync"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
	"github.com/docker/swarm/filter"
	"github.com/docker/swarm/strategy"
	"github.com/samalba/dockerclient"
)

var ErrNotImplemented = errors.New("not implemented in the mesos scheduler")

type MesosScheduler struct {
	sync.Mutex

	cluster  *cluster.Cluster
	strategy strategy.PlacementStrategy
	filters  []filter.Filter
}

func (s *MesosScheduler) Initialize(cluster *cluster.Cluster, strategy strategy.PlacementStrategy, filters []filter.Filter) {
	s.cluster = cluster
	s.strategy = strategy
	s.filters = filters
}

// Schedule a brand new container into the cluster.
func (s *MesosScheduler) CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error) {

	s.Lock()
	defer s.Unlock()

	//TODO: RequestOffers from mesos master

	//TODO: pick the right offer (using strategy & filters ???)

	//TODO: LaunchTask on the Mesos cluster and get container

	//TODO: Store container in store
	//  -   s.cluster.CommitContainerInStore(container.Id, config, name)

	return nil, ErrNotImplemented
}

// Remove a container from the cluster. Containers should always be destroyed
// through the scheduler to guarantee atomicity.
func (s *MesosScheduler) RemoveContainer(container *cluster.Container, force bool) error {
	s.Lock()
	defer s.Unlock()

	//TODO: KillTask

	//TODO: remove container
	//  -   s.cluster.RemoveContainerFromStore(container.Id, force)

	return ErrNotImplemented
}

// Entries are Mesos masters
func (s *MesosScheduler) NewEntries(entries []*discovery.Entry) {

	//TODO: get list of actual docker nodes from mesos masters
	//  -   cluster.NewNode(m.String(), s.cluster.OvercommitRatio)

	//TODO: create direct connection to those nodes
	//  -   n.Connect(s.cluster.TLSConfig)

	//TODO: add them to the cluster
	//  -   s.cluster.AddNode(n)
}

func (s *MesosScheduler) Events(eventsHandler cluster.EventHandler) {
	s.cluster.Events(eventsHandler)
}

func (s *MesosScheduler) Nodes() []*cluster.Node {
	return s.cluster.Nodes()
}

func (s *MesosScheduler) Containers() []*cluster.Container {
	return s.cluster.Containers()
}

func (s *MesosScheduler) Container(IdOrName string) *cluster.Container {
	return s.cluster.Container(IdOrName)
}
