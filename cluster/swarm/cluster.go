package swarm

import (
	"fmt"
	"sort"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/units"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/scheduler/node"
	"github.com/docker/swarm/state"
	"github.com/samalba/dockerclient"
)

// Cluster is exported
type Cluster struct {
	sync.RWMutex

	eventHandler cluster.EventHandler
	nodes        map[string]*cluster.Engine
	scheduler    *scheduler.Scheduler
	options      *cluster.Options
	store        *state.Store
}

// NewCluster is exported
func NewCluster(scheduler *scheduler.Scheduler, store *state.Store, eventhandler cluster.EventHandler, options *cluster.Options) cluster.Cluster {
	log.WithFields(log.Fields{"name": "swarm"}).Debug("Initializing cluster")

	cluster := &Cluster{
		eventHandler: eventhandler,
		nodes:        make(map[string]*cluster.Engine),
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
	c.scheduler.Lock()
	defer c.scheduler.Unlock()

	n, err := c.scheduler.SelectNodeForContainer(c.listNodes(), config)
	if err != nil {
		return nil, err
	}

	if nn, ok := c.nodes[n.ID]; ok {
		container, err := nn.Create(config, name, true)
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
	c.scheduler.Lock()
	defer c.scheduler.Unlock()

	if err := container.Engine.Destroy(container, force); err != nil {
		return err
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
			if !c.hasNode(m.String()) {
				n := cluster.NewEngine(m.String(), c.options.OvercommitRatio)
				if err := n.Connect(c.options.TLSConfig); err != nil {
					log.Error(err)
					return
				}
				c.Lock()

				if old, exists := c.nodes[n.ID]; exists {
					c.Unlock()
					if old.IP != n.IP {
						log.Errorf("ID duplicated. %s shared by %s and %s", n.ID, old.IP, n.IP)
					} else {
						log.Errorf("node %q is already registered", n.ID)
					}
					return
				}
				c.nodes[n.ID] = n
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

func (c *Cluster) hasNode(addr string) bool {
	for _, node := range c.nodes {
		if node.Addr == addr {
			return true
		}
	}
	return false
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
	return image.Engine.RemoveImage(image)
}

// Pull is exported
func (c *Cluster) Pull(name string, callback func(what, status string)) {
	size := len(c.nodes)
	done := make(chan bool, size)
	for _, n := range c.nodes {
		go func(nn *cluster.Engine) {
			if callback != nil {
				callback(nn.Name, "")
			}
			err := nn.Pull(name)
			if callback != nil {
				if err != nil {
					callback(nn.Name, err.Error())
				} else {
					callback(nn.Name, "downloaded")
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

// listNodes returns all the nodes in the cluster.
func (c *Cluster) listNodes() []*node.Node {
	c.RLock()
	defer c.RUnlock()

	out := []*node.Node{}
	for _, n := range c.nodes {
		out = append(out, node.NewNode(n))
	}

	return out
}

// listEngines returns all the engines in the cluster.
func (c *Cluster) listEngines() []*cluster.Engine {
	c.RLock()
	defer c.RUnlock()

	out := []*cluster.Engine{}
	for _, n := range c.nodes {
		out = append(out, n)
	}
	return out
}

// Info is exported
func (c *Cluster) Info() [][2]string {
	info := [][2]string{
		{"\bStrategy", c.scheduler.Strategy()},
		{"\bFilters", c.scheduler.Filters()},
		{"\bNodes", fmt.Sprintf("%d", len(c.nodes))},
	}

	nodes := c.listEngines()
	sort.Sort(cluster.EngineSorter(nodes))

	for _, node := range c.nodes {
		info = append(info, [2]string{node.Name, node.Addr})
		info = append(info, [2]string{" └ Containers", fmt.Sprintf("%d", len(node.Containers()))})
		info = append(info, [2]string{" └ Reserved CPUs", fmt.Sprintf("%d / %d", node.UsedCpus(), node.TotalCpus())})
		info = append(info, [2]string{" └ Reserved Memory", fmt.Sprintf("%s / %s", units.BytesSize(float64(node.UsedMemory())), units.BytesSize(float64(node.TotalMemory())))})
	}

	return info
}
