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

	eventHandler cluster.EventHandler
	nodes        map[string]*Node
	scheduler    *scheduler.Scheduler
	options      *cluster.Options
	store        *state.Store
}

func NewCluster(scheduler *scheduler.Scheduler, store *state.Store, eventhandler cluster.EventHandler, options *cluster.Options) cluster.Cluster {
	log.WithFields(log.Fields{"name": "swarm"}).Debug("Initializing cluster")

	cluster := &SwarmCluster{
		eventHandler: eventhandler,
		nodes:        make(map[string]*Node),
		scheduler:    scheduler,
		options:      options,
		store:        store,
	}

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

// callback for the events
func (s *SwarmCluster) Handle(e *cluster.Event) error {
	if err := s.eventHandler.Handle(e); err != nil {
		log.Error(err)
	}
	return nil
}

// Schedule a brand new container into the cluster.
func (s *SwarmCluster) CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error) {

	s.RLock()
	defer s.RUnlock()

	node, err := s.scheduler.SelectNodeForContainer(s.listNodes(), config)
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
			if s.getNode(m.String()) == nil {
				n := NewNode(m.String(), s.options.OvercommitRatio)
				if err := n.Connect(s.options.TLSConfig); err != nil {
					log.Error(err)
					return
				}
				s.Lock()

				if old, exists := s.nodes[n.id]; exists {
					s.Unlock()
					if old.ip != n.ip {
						log.Errorf("ID duplicated. %s shared by %s and %s", n.id, old.IP(), n.IP())
					} else {
						log.Errorf("node %q is already registered", n.id)
					}
					return
				}
				s.nodes[n.id] = n
				if err := n.Events(s); err != nil {
					log.Error(err)
					s.Unlock()
					return
				}
				s.Unlock()

			}
		}(entry)
	}
}

func (s *SwarmCluster) getNode(addr string) *Node {
	for _, node := range s.nodes {
		if node.addr == addr {
			return node
		}
	}
	return nil
}

// Containers returns all the images in the cluster.
func (s *SwarmCluster) Images() []*cluster.Image {
	s.RLock()
	defer s.RUnlock()

	out := []*cluster.Image{}
	for _, n := range s.nodes {
		out = append(out, n.Images()...)
	}

	return out
}

// Image returns an image with IdOrName in the cluster
func (s *SwarmCluster) Image(IdOrName string) *cluster.Image {
	// Abort immediately if the name is empty.
	if len(IdOrName) == 0 {
		return nil
	}

	s.RLock()
	defer s.RUnlock()
	for _, n := range s.nodes {
		if image := n.Image(IdOrName); image != nil {
			return image
		}
	}

	return nil
}

// Containers returns all the containers in the cluster.
func (s *SwarmCluster) Containers() []*cluster.Container {
	s.RLock()
	defer s.RUnlock()

	out := []*cluster.Container{}
	for _, n := range s.nodes {
		out = append(out, n.Containers()...)
	}

	return out
}

// Container returns the container with IdOrName in the cluster
func (s *SwarmCluster) Container(IdOrName string) *cluster.Container {
	// Abort immediately if the name is empty.
	if len(IdOrName) == 0 {
		return nil
	}

	s.RLock()
	defer s.RUnlock()
	for _, n := range s.nodes {
		if container := n.Container(IdOrName); container != nil {
			return container
		}
	}

	return nil
}

// nodes returns all the nodess in the cluster.
func (s *SwarmCluster) listNodes() []cluster.Node {
	s.RLock()
	defer s.RUnlock()

	out := []cluster.Node{}
	for _, n := range s.nodes {
		out = append(out, n)
	}

	return out
}

func (s *SwarmCluster) Info() [][2]string {
	info := [][2]string{{"\bNodes", fmt.Sprintf("%d", len(s.nodes))}}

	for _, node := range s.nodes {
		info = append(info, [2]string{node.Name(), node.Addr()})
		info = append(info, [2]string{" └ Containers", fmt.Sprintf("%d", len(node.Containers()))})
		info = append(info, [2]string{" └ Reserved CPUs", fmt.Sprintf("%d / %d", node.UsedCpus(), node.TotalCpus())})
		info = append(info, [2]string{" └ Reserved Memory", fmt.Sprintf("%s / %s", units.BytesSize(float64(node.UsedMemory())), units.BytesSize(float64(node.TotalMemory())))})
	}

	return info
}
