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

// Cluster is exported
type Cluster struct {
	sync.RWMutex

	eventHandler cluster.EventHandler
	nodes        map[string]*node
	scheduler    *scheduler.Scheduler
	options      *cluster.Options
	store        *state.Store
}

// NewCluster is exported
func NewCluster(scheduler *scheduler.Scheduler, store *state.Store, eventhandler cluster.EventHandler, options *cluster.Options) cluster.Cluster {
	log.WithFields(log.Fields{"name": "swarm"}).Debug("Initializing cluster")

	cluster := &Cluster{
		eventHandler: eventhandler,
		nodes:        make(map[string]*node),
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

// Handle callbacks for the events
func (c *Cluster) Handle(e *cluster.Event) error {
	if err := c.eventHandler.Handle(e); err != nil {
		log.Error(err)
	}
	return nil
}

// CreateContainer aka schedule a brand new container into the cluster.
func (c *Cluster) CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error) {

	c.RLock()
	defer c.RUnlock()
retry:
	// FIXME: to prevent a race, we check again after the pull if the node can still handle
	// the container. We should store the state in the store before pulling and use this to check
	// all the other container create, but, as we don't have a proper store yet, this temporary solution
	// was chosen.
	n, err := c.scheduler.SelectNodeForContainer(c.listNodes(), config)
	if err != nil {
		return nil, err
	}

	if nn, ok := n.(*node); ok {
		container, err := nn.create(config, name, false)
		if err == dockerclient.ErrNotFound {
			// image not on the node, try to pull
			if err = nn.Pull(config.Image); err != nil {
				return nil, err
			}

			// check if the container can still fit on this node
			if _, err = c.scheduler.SelectNodeForContainer([]cluster.Node{n}, config); err != nil {
				// if not, try to find another node
				log.Debugf("Node %s not available anymore, selecting another one", n.Name())
				goto retry
			}
			container, err = nn.create(config, name, false)
		}
		if err != nil {
			return nil, err
		}

		st := &state.RequestedState{
			ID:     container.Id,
			Name:   name,
			Config: config,
		}
		return container, c.store.Add(container.Id, st)
	}

	return nil, nil
}

// RemoveContainer aka Remove a container from the cluster. Containers should
// always be destroyed through the scheduler to guarantee atomicity.
func (c *Cluster) RemoveContainer(container *cluster.Container, force bool) error {
	c.Lock()
	defer c.Unlock()

	if n, ok := container.Node.(*node); ok {
		if err := n.destroy(container, force); err != nil {
			return err
		}
	}

	if err := c.store.Remove(container.Id); err != nil {
		if err == state.ErrNotFound {
			log.Debugf("Container %s not found in the store", container.Id)
			return nil
		}
		return err
	}
	return nil
}

// Entries are Docker Nodes
func (c *Cluster) newEntries(entries []*discovery.Entry) {
	for _, entry := range entries {
		go func(m *discovery.Entry) {
			if c.getNode(m.String()) == nil {
				n := NewNode(m.String(), c.options.OvercommitRatio)
				if err := n.Connect(c.options.TLSConfig); err != nil {
					log.Error(err)
					return
				}
				c.Lock()

				if old, exists := c.nodes[n.ID()]; exists {
					c.Unlock()
					if old.IP() != n.IP() {
						log.Errorf("ID duplicated. %s shared by %s and %s", n.ID(), old.IP(), n.IP())
					} else {
						log.Errorf("node %q is already registered", n.ID())
					}
					return
				}
				c.nodes[n.ID()] = n
				if err := n.Events(c); err != nil {
					log.Error(err)
					c.Unlock()
					return
				}
				c.Unlock()

			}
		}(entry)
	}
}

func (c *Cluster) getNode(addr string) *node {
	for _, node := range c.nodes {
		if node.Addr() == addr {
			return node
		}
	}
	return nil
}

// Images returns all the images in the cluster.
func (c *Cluster) Images() []*cluster.Image {
	c.RLock()
	defer c.RUnlock()

	out := []*cluster.Image{}
	for _, n := range c.nodes {
		out = append(out, n.Images()...)
	}

	return out
}

// Image returns an image with IDOrName in the cluster
func (c *Cluster) Image(IDOrName string) *cluster.Image {
	// Abort immediately if the name is empty.
	if len(IDOrName) == 0 {
		return nil
	}

	c.RLock()
	defer c.RUnlock()
	for _, n := range c.nodes {
		if image := n.Image(IDOrName); image != nil {
			return image
		}
	}

	return nil
}

// RemoveImage removes an image from the cluster
func (c *Cluster) RemoveImage(image *cluster.Image) ([]*dockerclient.ImageDelete, error) {
	c.Lock()
	defer c.Unlock()
	if n, ok := image.Node.(*node); ok {
		return n.RemoveImage(image)
	}
	return nil, nil
}

// Pull is exported
func (c *Cluster) Pull(name string, callback func(what, status string)) {
	size := len(c.nodes)
	done := make(chan bool, size)
	for _, n := range c.nodes {
		go func(nn *node) {
			if callback != nil {
				callback(nn.Name(), "")
			}
			err := nn.Pull(name)
			if callback != nil {
				if err != nil {
					callback(nn.Name(), err.Error())
				} else {
					callback(nn.Name(), "downloaded")
				}
			}
			done <- true
		}(n)
	}
	for i := 0; i < size; i++ {
		<-done
	}
}

// Containers returns all the containers in the cluster.
func (c *Cluster) Containers() []*cluster.Container {
	c.RLock()
	defer c.RUnlock()

	out := []*cluster.Container{}
	for _, n := range c.nodes {
		out = append(out, n.Containers()...)
	}

	return out
}

// Container returns the container with IDOrName in the cluster
func (c *Cluster) Container(IDOrName string) *cluster.Container {
	// Abort immediately if the name is empty.
	if len(IDOrName) == 0 {
		return nil
	}

	c.RLock()
	defer c.RUnlock()
	for _, n := range c.nodes {
		if container := n.Container(IDOrName); container != nil {
			return container
		}
	}

	return nil
}

// nodes returns all the nodess in the cluster.
func (c *Cluster) listNodes() []cluster.Node {
	c.RLock()
	defer c.RUnlock()

	out := []cluster.Node{}
	for _, n := range c.nodes {
		out = append(out, n)
	}

	return out
}

// Info is exported
func (c *Cluster) Info() [][2]string {
	info := [][2]string{{"\bNodes", fmt.Sprintf("%d", len(c.nodes))}}

	for _, node := range c.nodes {
		info = append(info, [2]string{node.Name(), node.Addr()})
		info = append(info, [2]string{" └ Containers", fmt.Sprintf("%d", len(node.Containers()))})
		info = append(info, [2]string{" └ Reserved CPUs", fmt.Sprintf("%d / %d", node.UsedCpus(), node.TotalCpus())})
		info = append(info, [2]string{" └ Reserved Memory", fmt.Sprintf("%s / %s", units.BytesSize(float64(node.UsedMemory())), units.BytesSize(float64(node.TotalMemory())))})
	}

	return info
}
