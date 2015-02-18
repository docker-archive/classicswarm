package swarm

import (
	"fmt"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/units"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/state"
	"github.com/samalba/dockerclient"
)

type SwarmCluster struct {
	sync.RWMutex

	nodes     *Nodes
	scheduler *scheduler.Scheduler
	options   *cluster.Options
	store     *state.Store
}

func NewCluster(scheduler *scheduler.Scheduler, store *state.Store, options *cluster.Options) cluster.Cluster {
	log.WithFields(log.Fields{"name": "swarm"}).Debug("Initializing cluster")

	cluster := &SwarmCluster{
		nodes:     NewNodes(),
		scheduler: scheduler,
		options:   options,
		store:     store,
	}

	cluster.nodes.Events(options.EventsHandler)

	// get the list of entries from the discovery service
	go func() {
		d, err := discovery.New(options.Discovery, options.Heartbeat)
		if err != nil {
			log.Fatal(err)
		}

		entries, err := d.Fetch()
		if err != nil {
			log.Fatal(err)

		}
		cluster.newEntries(entries)

		go d.Watch(cluster.newEntries)
	}()

	return cluster
}

// Schedule a brand new container into the cluster.
func (s *SwarmCluster) CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error) {

	s.RLock()
	defer s.RUnlock()

	node, err := s.scheduler.SelectNodeForContainer(s.nodes.List(), config)
	if err != nil {
		return nil, err
	}

	if n, ok := node.(*Node); ok {
		container, err := n.Create(config, name, true)
		if err != nil {
			return nil, err
		}

		st := &state.RequestedState{
			ID:     container.Id,
			Name:   name,
			Config: config,
		}
		return container, s.store.Add(container.Id, st)
	}

	return nil, nil
}

// Remove a container from the cluster. Containers should always be destroyed
// through the scheduler to guarantee atomicity.
func (s *SwarmCluster) RemoveContainer(container *cluster.Container, force bool) error {
	s.Lock()
	defer s.Unlock()

	if n, ok := container.Node.(*Node); ok {
		if err := n.Destroy(container, force); err != nil {
			return err
		}
	}

	if err := s.store.Remove(container.Id); err != nil {
		if err == state.ErrNotFound {
			log.Debugf("Container %s not found in the store", container.Id)
			return nil
		}
		return err
	}
	return nil
}

// Entries are Docker Nodes
func (s *SwarmCluster) newEntries(entries []*discovery.Entry) {
	for _, entry := range entries {
		go func(m *discovery.Entry) {
			if s.nodes.Get(m.String()) == nil {
				n := NewNode(m.String(), s.options.OvercommitRatio)
				if err := n.Connect(s.options.TLSConfig); err != nil {
					log.Error(err)
					return
				}
				if err := s.nodes.Add(n); err != nil {
					log.Error(err)
					return
				}
			}
		}(entry)
	}
}

func (s *SwarmCluster) Images() []*cluster.Image {
	return s.nodes.Images()
}

func (s *SwarmCluster) Image(IdOrName string) *cluster.Image {
	return s.nodes.Image(IdOrName)
}

func (s *SwarmCluster) Containers() []*cluster.Container {
	return s.nodes.Containers()
}

func (s *SwarmCluster) Container(IdOrName string) *cluster.Container {
	return s.nodes.Container(IdOrName)
}

func (s *SwarmCluster) Info() [][2]string {
	info := [][2]string{{"\bNodes", fmt.Sprintf("%d", len(s.nodes.List()))}}

	for _, node := range s.nodes.List() {
		info = append(info, [2]string{node.Name(), node.Addr()})
		info = append(info, [2]string{" └ Containers", fmt.Sprintf("%d", len(node.Containers()))})
		info = append(info, [2]string{" └ Reserved CPUs", fmt.Sprintf("%d / %d", node.UsedCpus(), node.TotalCpus())})
		info = append(info, [2]string{" └ Reserved Memory", fmt.Sprintf("%s / %s", units.BytesSize(float64(node.UsedMemory())), units.BytesSize(float64(node.TotalMemory())))})
	}

	return info
}
