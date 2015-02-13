package mesos

import (
	"errors"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
	"github.com/samalba/dockerclient"
)

var ErrNotImplemented = errors.New("not implemented in the mesos cluster")

type MesosCluster struct {
	sync.Mutex

	//TODO: list of mesos masters
	//TODO: list of offers
	nodes   *cluster.Nodes
	options *cluster.Options
}

func NewCluster(options *cluster.Options) cluster.Cluster {
	log.WithFields(log.Fields{"name": "mesos"}).Debug("Initializing cluster")
	return &MesosCluster{
		nodes:   cluster.NewNodes(),
		options: options,
	}
}

// Schedule a brand new container into the cluster.
func (s *MesosCluster) CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error) {

	s.Lock()
	defer s.Unlock()

	//TODO: pick the right offer (using strategy & filters ???)

	//TODO: LaunchTask on the Mesos cluster and get container

	//TODO: Store container in store ??

	return nil, ErrNotImplemented
}

// Remove a container from the cluster. Containers should always be destroyed
// through the scheduler to guarantee atomicity.
func (s *MesosCluster) RemoveContainer(container *cluster.Container, force bool) error {
	s.Lock()
	defer s.Unlock()

	//TODO: KillTask

	//TODO: remove container from store ??

	return ErrNotImplemented
}

// Entries are Mesos masters
func (s *MesosCluster) NewEntries(entries []*discovery.Entry) {

	//TODO: get list of actual docker nodes from mesos masters
	//  -   cluster.NewNode(m.String(), s.options.OvercommitRatio)

	//TODO: create direct connection to those nodes
	//  -   n.Connect(s.options.TLSConfig)

	//TODO: add them to the cluster
	//  -   s.nodes.Add(n)
}

func (s *MesosCluster) Events(eventsHandler cluster.EventHandler) {
	s.nodes.Events(eventsHandler)
}

func (s *MesosCluster) Nodes() []*cluster.Node {
	return s.nodes.List()
}

func (s *MesosCluster) Containers() []*cluster.Container {
	return s.nodes.Containers()
}

func (s *MesosCluster) Container(IdOrName string) *cluster.Container {
	return s.nodes.Container(IdOrName)
}
