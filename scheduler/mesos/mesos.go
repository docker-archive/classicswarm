package mesos

import (
	"errors"
	"sync"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
	"github.com/docker/swarm/scheduler/options"
	"github.com/samalba/dockerclient"
)

var ErrNotImplemented = errors.New("not implemented in the mesos scheduler")

type MesosScheduler struct {
	sync.Mutex

	//TODO: list of mesos masters
	cluster *cluster.Cluster
	options *options.SchedulerOptions
}

func (s *MesosScheduler) Initialize(options *options.SchedulerOptions) {
	s.options = options

	s.cluster = cluster.NewCluster(s.options.Store)
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
	//  -   cluster.NewNode(m.String(), s.options.OvercommitRatio)

	//TODO: create direct connection to those nodes
	//  -   n.Connect(s.options.TLSConfig)

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
