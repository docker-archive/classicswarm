package mesos

import (
	"errors"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
	"github.com/docker/swarm/filter"
	"github.com/docker/swarm/strategy"
	"github.com/samalba/dockerclient"
)

var ErrNotImplemented = errors.New("not implemented in the mesos scheduler")

type MesosScheduler struct {
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

	//TODO: RequestOffers from mesos master

	//TODO: pick the right offer (using strategy & filters ???)

	//TODO: LaunchTask on the Mesos cluster

	return nil, ErrNotImplemented
}

// Remove a container from the cluster. Containers should always be destroyed
// through the scheduler to guarantee atomicity.
func (s *MesosScheduler) RemoveContainer(container *cluster.Container, force bool) error {

	//TODO: KillTask

	//TODO: remove container

	return ErrNotImplemented
}

// Entries are Mesos masters
func (s *MesosScheduler) NewEntries(entries []*discovery.Entry) {

	//TODO: get list of actual docker nodes from mesos masters

	//TODO: create direct connection to those nodes

	//TODO: add them to the cluster
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
