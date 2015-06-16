package swarm

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/stringid"
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
	engines      map[string]*cluster.Engine
	scheduler    *scheduler.Scheduler
	store        *state.Store
	discovery    discovery.Discovery

	overcommitRatio float64
	TLSConfig       *tls.Config
}

// NewCluster is exported
func NewCluster(scheduler *scheduler.Scheduler, store *state.Store, TLSConfig *tls.Config, discovery discovery.Discovery, options cluster.DriverOpts) (cluster.Cluster, error) {
	log.WithFields(log.Fields{"name": "swarm"}).Debug("Initializing cluster")

	cluster := &Cluster{
		engines:         make(map[string]*cluster.Engine),
		scheduler:       scheduler,
		store:           store,
		TLSConfig:       TLSConfig,
		discovery:       discovery,
		overcommitRatio: 0.05,
	}

	if val, ok := options.Float("swarm.overcommit", ""); ok {
		cluster.overcommitRatio = val
	}

	discoveryCh, errCh := cluster.discovery.Watch(nil)
	go cluster.monitorDiscovery(discoveryCh, errCh)

	return cluster, nil
}

// Handle callbacks for the events
func (c *Cluster) Handle(e *cluster.Event) error {
	if c.eventHandler == nil {
		return nil
	}
	if err := c.eventHandler.Handle(e); err != nil {
		log.Error(err)
	}
	return nil
}

// RegisterEventHandler registers an event handler.
func (c *Cluster) RegisterEventHandler(h cluster.EventHandler) error {
	if c.eventHandler != nil {
		return errors.New("event handler already set")
	}
	c.eventHandler = h
	return nil
}

// Generate a globally (across the cluster) unique ID.
func (c *Cluster) generateUniqueID() string {
	for {
		id := stringid.GenerateRandomID()
		if c.Container(id) == nil {
			return id
		}
	}
}

// CreateContainer aka schedule a brand new container into the cluster.
func (c *Cluster) CreateContainer(config *cluster.ContainerConfig, name string) (*cluster.Container, error) {
	container, err := c.createContainer(config, name, false)

	//  fails with image not found, then try to reschedule with soft-image-affinity
	if err != nil && strings.HasSuffix(err.Error(), "not found") {
		// Check if the image exists in the cluster
		// If exists, retry with a soft-image-affinity
		if image := c.Image(config.Image); image != nil {
			container, err = c.createContainer(config, name, true)
		}
	}
	return container, err
}

func (c *Cluster) createContainer(config *cluster.ContainerConfig, name string, withSoftImageAffinity bool) (*cluster.Container, error) {
	c.scheduler.Lock()
	defer c.scheduler.Unlock()

	// Ensure the name is avaliable
	if cID := c.getIDFromName(name); cID != "" {
		return nil, fmt.Errorf("Conflict, The name %s is already assigned to %s. You have to delete (or rename) that container to be able to assign %s to a container again.", name, cID, name)
	}

	// Associate a Swarm ID to the container we are creating.
	config.SetSwarmID(c.generateUniqueID())

	configTemp := config
	if withSoftImageAffinity {
		configTemp.AddSoftImageAffinity(config.Image)
	}

	n, err := c.scheduler.SelectNodeForContainer(c.listNodes(), configTemp)
	if err != nil {
		return nil, err
	}

	if nn, ok := c.engines[n.ID]; ok {
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

	if err := container.Engine.RemoveContainer(container, force); err != nil {
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

func (c *Cluster) getEngineByAddr(addr string) *cluster.Engine {
	c.RLock()
	defer c.RUnlock()

	for _, engine := range c.engines {
		if engine.Addr == addr {
			return engine
		}
	}
	return nil
}

func (c *Cluster) hasEngineByAddr(addr string) bool {
	return c.getEngineByAddr(addr) != nil
}

func (c *Cluster) addEngine(addr string) bool {
	// Check the engine is already registered by address.
	if c.hasEngineByAddr(addr) {
		return false
	}

	// Attempt a connection to the engine. Since this is slow, don't get a hold
	// of the lock yet.
	engine := cluster.NewEngine(addr, c.overcommitRatio)
	if err := engine.Connect(c.TLSConfig); err != nil {
		log.Error(err)
		return false
	}

	// The following is critical and fast. Grab a lock.
	c.Lock()
	defer c.Unlock()

	// Make sure the engine ID is unique.
	if old, exists := c.engines[engine.ID]; exists {
		if old.Addr != engine.Addr {
			log.Errorf("ID duplicated. %s shared by %s and %s", engine.ID, old.Addr, engine.Addr)
		} else {
			log.Debugf("node %q (name: %q) with address %q is already registered", engine.ID, engine.Name, engine.Addr)
		}
		engine.Disconnect()
		return false
	}

	// Finally register the engine.
	c.engines[engine.ID] = engine
	if err := engine.RegisterEventHandler(c); err != nil {
		log.Error(err)
	}

	log.Infof("Registered Engine %s at %s", engine.Name, addr)
	return true
}

func (c *Cluster) removeEngine(addr string) bool {
	engine := c.getEngineByAddr(addr)
	if engine == nil {
		return false
	}
	c.Lock()
	defer c.Unlock()

	engine.Disconnect()
	delete(c.engines, engine.ID)
	log.Infof("Removed Engine %s", engine.Name)
	return true
}

// Entries are Docker Engines
func (c *Cluster) monitorDiscovery(ch <-chan discovery.Entries, errCh <-chan error) {
	// Watch changes on the discovery channel.
	currentEntries := discovery.Entries{}
	for {
		select {
		case entries := <-ch:
			added, removed := currentEntries.Diff(entries)
			currentEntries = entries

			// Remove engines first. `addEngine` will refuse to add an engine
			// if there's already an engine with the same ID.  If an engine
			// changes address, we have to first remove it then add it back.
			for _, entry := range removed {
				c.removeEngine(entry.String())
			}

			// Since `addEngine` can be very slow (it has to connect to the
			// engine), we are going to do the adds in parallel.
			for _, entry := range added {
				go c.addEngine(entry.String())
			}
		case err := <-errCh:
			log.Errorf("Discovery error: %v", err)
		}
	}
}

// Images returns all the images in the cluster.
func (c *Cluster) Images() []*cluster.Image {
	c.RLock()
	defer c.RUnlock()

	out := []*cluster.Image{}
	for _, n := range c.engines {
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
	for _, n := range c.engines {
		if image := n.Image(IDOrName); image != nil {
			return image
		}
	}

	return nil
}

// RemoveImages removes all the images that match `name` from the cluster
func (c *Cluster) RemoveImages(name string) ([]*dockerclient.ImageDelete, error) {
	c.Lock()
	defer c.Unlock()

	out := []*dockerclient.ImageDelete{}
	errs := []string{}
	var err error
	for _, n := range c.engines {
		for _, image := range n.Images() {
			if image.Match(name, true) {
				content, err := image.Engine.RemoveImage(image, name)
				if err != nil {
					errs = append(errs, fmt.Sprintf("%s: %s", image.Engine.Name, err.Error()))
					continue
				}
				out = append(out, content...)
			}
		}
	}

	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, "\n"))
	}

	return out, err
}

// Pull is exported
func (c *Cluster) Pull(name string, authConfig *dockerclient.AuthConfig, callback func(what, status string)) {
	var wg sync.WaitGroup

	c.RLock()
	for _, n := range c.engines {
		wg.Add(1)

		go func(nn *cluster.Engine) {
			defer wg.Done()

			if callback != nil {
				callback(nn.Name, "")
			}
			err := nn.Pull(name, authConfig)
			if callback != nil {
				if err != nil {
					callback(nn.Name, err.Error())
				} else {
					callback(nn.Name, "downloaded")
				}
			}
		}(n)
	}
	c.RUnlock()

	wg.Wait()
}

// Load image
func (c *Cluster) Load(imageReader io.Reader, callback func(what, status string)) {
	var wg sync.WaitGroup

	c.RLock()
	pipeWriters := []*io.PipeWriter{}
	for _, n := range c.engines {
		wg.Add(1)

		pipeReader, pipeWriter := io.Pipe()
		pipeWriters = append(pipeWriters, pipeWriter)

		go func(reader *io.PipeReader, nn *cluster.Engine) {
			defer wg.Done()
			defer reader.Close()

			// call engine load image
			err := nn.Load(reader)
			if callback != nil {
				if err != nil {
					callback(nn.Name, err.Error())
				}
			}
		}(pipeReader, n)
	}
	c.RUnlock()

	// create multi-writer
	listWriter := []io.Writer{}
	for _, pipeW := range pipeWriters {
		listWriter = append(listWriter, pipeW)
	}
	multiWriter := io.MultiWriter(listWriter...)

	// copy image-reader to multi-writer
	_, err := io.Copy(multiWriter, imageReader)
	if err != nil {
		log.Error(err)
	}

	// close pipe writers
	for _, pipeW := range pipeWriters {
		pipeW.Close()
	}

	wg.Wait()
}

// Import image
func (c *Cluster) Import(source string, repository string, tag string, imageReader io.Reader, callback func(what, status string)) {
	var wg sync.WaitGroup
	c.RLock()
	pipeWriters := []*io.PipeWriter{}

	for _, n := range c.engines {
		wg.Add(1)

		pipeReader, pipeWriter := io.Pipe()
		pipeWriters = append(pipeWriters, pipeWriter)

		go func(reader *io.PipeReader, nn *cluster.Engine) {
			defer wg.Done()
			defer reader.Close()

			// call engine import
			err := nn.Import(source, repository, tag, reader)
			if callback != nil {
				if err != nil {
					callback(nn.Name, err.Error())
				} else {
					callback(nn.Name, "Import success")
				}
			}

		}(pipeReader, n)
	}
	c.RUnlock()

	// create multi-writer
	listWriter := []io.Writer{}
	for _, pipeW := range pipeWriters {
		listWriter = append(listWriter, pipeW)
	}
	multiWriter := io.MultiWriter(listWriter...)

	// copy image-reader to muti-writer
	_, err := io.Copy(multiWriter, imageReader)
	if err != nil {
		log.Error(err)
	}

	// close pipe writers
	for _, pipeW := range pipeWriters {
		pipeW.Close()
	}

	wg.Wait()
}

// Containers returns all the containers in the cluster.
func (c *Cluster) Containers() cluster.Containers {
	c.RLock()
	defer c.RUnlock()

	out := cluster.Containers{}
	for _, n := range c.engines {
		out = append(out, n.Containers()...)
	}

	return out
}

func (c *Cluster) getIDFromName(name string) string {
	// Abort immediately if the name is empty.
	if len(name) == 0 {
		return ""
	}

	c.RLock()
	defer c.RUnlock()
	for _, n := range c.engines {
		for _, c := range n.Containers() {
			for _, cname := range c.Names {
				if cname == name || cname == "/"+name {
					return c.Id
				}
			}
		}
	}
	return ""
}

// Container returns the container with IDOrName in the cluster
func (c *Cluster) Container(IDOrName string) *cluster.Container {
	// Abort immediately if the name is empty.
	if len(IDOrName) == 0 {
		return nil
	}

	c.RLock()
	defer c.RUnlock()

	return cluster.Containers(c.Containers()).Get(IDOrName)

}

// listNodes returns all the engines in the cluster.
func (c *Cluster) listNodes() []*node.Node {
	c.RLock()
	defer c.RUnlock()

	out := make([]*node.Node, 0, len(c.engines))
	for _, n := range c.engines {
		out = append(out, node.NewNode(n))
	}

	return out
}

// listEngines returns all the engines in the cluster.
func (c *Cluster) listEngines() []*cluster.Engine {
	c.RLock()
	defer c.RUnlock()

	out := make([]*cluster.Engine, 0, len(c.engines))
	for _, n := range c.engines {
		out = append(out, n)
	}
	return out
}

// TotalMemory return the total memory of the cluster
func (c *Cluster) TotalMemory() int64 {
	var totalMemory int64
	for _, engine := range c.engines {
		totalMemory += engine.TotalMemory()
	}
	return totalMemory
}

// TotalCpus return the total memory of the cluster
func (c *Cluster) TotalCpus() int64 {
	var totalCpus int64
	for _, engine := range c.engines {
		totalCpus += engine.TotalCpus()
	}
	return totalCpus
}

// Info returns some info about the cluster, like nb or containers / images
func (c *Cluster) Info() [][]string {
	info := [][]string{
		{"\bStrategy", c.scheduler.Strategy()},
		{"\bFilters", c.scheduler.Filters()},
		{"\bNodes", fmt.Sprintf("%d", len(c.engines))},
	}

	engines := c.listEngines()
	sort.Sort(cluster.EngineSorter(engines))

	for _, engine := range engines {
		info = append(info, []string{engine.Name, engine.Addr})
		info = append(info, []string{" └ Containers", fmt.Sprintf("%d", len(engine.Containers()))})
		info = append(info, []string{" └ Reserved CPUs", fmt.Sprintf("%d / %d", engine.UsedCpus(), engine.TotalCpus())})
		info = append(info, []string{" └ Reserved Memory", fmt.Sprintf("%s / %s", units.BytesSize(float64(engine.UsedMemory())), units.BytesSize(float64(engine.TotalMemory())))})
		labels := make([]string, 0, len(engine.Labels))
		for k, v := range engine.Labels {
			labels = append(labels, k+"="+v)
		}
		sort.Strings(labels)
		info = append(info, []string{" └ Labels", fmt.Sprintf("%s", strings.Join(labels, ", "))})
	}

	return info
}

// RANDOMENGINE returns a random engine.
func (c *Cluster) RANDOMENGINE() (*cluster.Engine, error) {
	n, err := c.scheduler.SelectNodeForContainer(c.listNodes(), &cluster.ContainerConfig{})
	if err != nil {
		return nil, err
	}
	if n != nil {
		return c.engines[n.ID], nil
	}
	return nil, nil
}

// RenameContainer rename a container
func (c *Cluster) RenameContainer(container *cluster.Container, newName string) error {
	c.RLock()
	defer c.RUnlock()

	// check new name whether avaliable
	if cID := c.getIDFromName(newName); cID != "" {
		return fmt.Errorf("Conflict, The name %s is already assigned to %s. You have to delete (or rename) that container to be able to assign %s to a container again.", newName, cID, newName)
	}

	// call engine rename
	err := container.Engine.RenameContainer(container, newName)
	if err != nil {
		return err
	}

	// update container name in store
	st, err := c.store.Get(container.Id)
	if err != nil {
		return err
	}
	st.Name = newName
	return c.store.Replace(container.Id, st)
}
