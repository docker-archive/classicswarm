package mesos

import (
	"errors"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler"
	"github.com/samalba/dockerclient"
)

var ErrNotImplemented = errors.New("not implemented in the mesos cluster")

type MesosCluster struct {
	sync.Mutex

	//TODO: list of mesos masters
	//TODO: list of offers
	scheduler *scheduler.Scheduler
	options   *cluster.Options
}

func NewCluster(scheduler *scheduler.Scheduler, options *cluster.Options) cluster.Cluster {
	log.WithFields(log.Fields{"name": "mesos"}).Debug("Initializing cluster")

	//TODO: get the list of mesos masters using options.Discovery (zk://<ip1>,<ip2>,<ip3>/mesos)

	return &MesosCluster{
		scheduler: scheduler,
		options:   options,
	}
}

// Schedule a brand new container into the cluster.
func (s *MesosCluster) CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error) {

	s.Lock()
	defer s.Unlock()

	//TODO: pick the right offer (using strategy & filters)
	//offer, err := s.scheduler.SelectNodeForContainer(s.offers, config)

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

func (s *MesosCluster) Images() []*cluster.Image {
	return nil
}

func (s *MesosCluster) Image(IdOrName string) *cluster.Image {
	return nil
}

func (s *MesosCluster) Containers() []*cluster.Container {
	return nil
}

func (s *MesosCluster) Container(IdOrName string) *cluster.Container {
	return nil
}

func (s *MesosCluster) Info() [][2]string {
	return nil
}
