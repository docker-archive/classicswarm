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
	dockerfilters "github.com/docker/docker/pkg/parsers/filters"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/pkg/units"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/discovery"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/scheduler/node"
	"github.com/samalba/dockerclient"
)

type pendingContainer struct {
	Config *cluster.ContainerConfig
	Name   string
	Engine *cluster.Engine
}

func (p *pendingContainer) ToContainer() *cluster.Container {
	container := &cluster.Container{
		Container: dockerclient.Container{},
		Config:    p.Config,
		Info: dockerclient.ContainerInfo{
			HostConfig: &dockerclient.HostConfig{},
		},
		Engine: p.Engine,
	}

	if p.Name != "" {
		container.Container.Names = []string{"/" + p.Name}
	}

	return container
}

// Cluster is exported
type Cluster struct {
	sync.RWMutex

	eventHandler      cluster.EventHandler
	engines           map[string]*cluster.Engine
	scheduler         *scheduler.Scheduler
	discovery         discovery.Discovery
	pendingContainers map[string]*pendingContainer

	overcommitRatio float64
	TLSConfig       *tls.Config
}

// NewCluster is exported
func NewCluster(scheduler *scheduler.Scheduler, TLSConfig *tls.Config, discovery discovery.Discovery, options cluster.DriverOpts) (cluster.Cluster, error) {
	log.WithFields(log.Fields{"name": "swarm"}).Debug("Initializing cluster")

	cluster := &Cluster{
		engines:           make(map[string]*cluster.Engine),
		scheduler:         scheduler,
		TLSConfig:         TLSConfig,
		discovery:         discovery,
		pendingContainers: make(map[string]*pendingContainer),
		overcommitRatio:   0.05,
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
	if err != nil && strings.HasSuffix(err.Error(), "not found") && !config.HaveNodeConstraint() {
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

	// Ensure the name is available
	if !c.checkNameUniqueness(name) {
		c.scheduler.Unlock()
		return nil, fmt.Errorf("Conflict: The name %s is already assigned. You have to delete (or rename) that container to be able to assign %s to a container again.", name, name)
	}

	// Associate a Swarm ID to the container we are creating.
	swarmID := c.generateUniqueID()
	config.SetSwarmID(swarmID)

	configTemp := config
	if withSoftImageAffinity {
		configTemp.AddAffinity("image==~" + config.Image)
	}

	nodes, err := c.scheduler.SelectNodesForContainer(c.listNodes(), configTemp)
	if err != nil {
		c.scheduler.Unlock()
		return nil, err
	}
	n := nodes[0]
	engine, ok := c.engines[n.ID]
	if !ok {
		c.scheduler.Unlock()
		return nil, fmt.Errorf("error creating container")
	}

	c.pendingContainers[swarmID] = &pendingContainer{
		Name:   name,
		Config: config,
		Engine: engine,
	}

	c.scheduler.Unlock()

	container, err := engine.Create(config, name, true)

	c.scheduler.Lock()
	delete(c.pendingContainers, swarmID)
	c.scheduler.Unlock()

	return container, err
}

// RemoveContainer aka Remove a container from the cluster.
func (c *Cluster) RemoveContainer(container *cluster.Container, force, volumes bool) error {
	return container.Engine.RemoveContainer(container, force, volumes)
}

// RemoveNetwork removes a network from the cluster
func (c *Cluster) RemoveNetwork(network *cluster.Network) error {
	err := network.Engine.RemoveNetwork(network)
	c.refreshNetworks()
	return err
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

	engine := cluster.NewEngine(addr, c.overcommitRatio)
	if err := engine.RegisterEventHandler(c); err != nil {
		log.Error(err)
	}

	// Attempt a connection to the engine. Since this is slow, don't get a hold
	// of the lock yet.
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
func (c *Cluster) Images(all bool, filters dockerfilters.Args) []*cluster.Image {
	c.RLock()
	defer c.RUnlock()

	out := []*cluster.Image{}
	for _, e := range c.engines {
		out = append(out, e.Images(all, filters)...)
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
	for _, e := range c.engines {
		if image := e.Image(IDOrName); image != nil {
			return image
		}
	}

	return nil
}

// RemoveImages removes all the images that match `name` from the cluster
func (c *Cluster) RemoveImages(name string, force bool) ([]*dockerclient.ImageDelete, error) {
	c.Lock()
	defer c.Unlock()

	out := []*dockerclient.ImageDelete{}
	errs := []string{}
	var err error
	for _, e := range c.engines {
		for _, image := range e.Images(true, nil) {
			if image.Match(name, true) {
				content, err := image.Engine.RemoveImage(image, name, force)
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

func (c *Cluster) refreshNetworks() {
	var wg sync.WaitGroup
	for _, e := range c.engines {
		wg.Add(1)
		go func(e *cluster.Engine) {
			e.RefreshNetworks()
			wg.Done()
		}(e)
	}
	wg.Wait()
}

// CreateNetwork creates a network in the cluster
func (c *Cluster) CreateNetwork(request *dockerclient.NetworkCreate) (response *dockerclient.NetworkCreateResponse, err error) {
	var (
		parts  = strings.SplitN(request.Name, "/", 2)
		config = &cluster.ContainerConfig{}
	)

	if len(parts) == 2 {
		// a node was specified, create the container only on this node
		request.Name = parts[1]
		config = cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:node==" + parts[0]}})
	}

	nodes, err := c.scheduler.SelectNodesForContainer(c.listNodes(), config)
	if err != nil {
		return nil, err
	}
	if nodes != nil {
		resp, err := c.engines[nodes[0].ID].CreateNetwork(request)
		c.refreshNetworks()
		return resp, err
	}
	return nil, nil
}

// CreateVolume creates a volume in the cluster
func (c *Cluster) CreateVolume(request *dockerclient.VolumeCreateRequest) (*cluster.Volume, error) {
	var (
		wg     sync.WaitGroup
		volume *cluster.Volume
		err    error
	)

	if request.Name == "" {
		request.Name = stringid.GenerateRandomID()
	}

	c.RLock()
	for _, e := range c.engines {
		wg.Add(1)

		go func(engine *cluster.Engine) {
			defer wg.Done()

			v, er := engine.CreateVolume(request)
			if v != nil {
				volume = v
				err = nil
			}
			if er != nil && volume == nil {
				err = er
			}
		}(e)
	}
	c.RUnlock()

	wg.Wait()

	return volume, err
}

// RemoveVolumes removes all the volumes that match `name` from the cluster
func (c *Cluster) RemoveVolumes(name string) (bool, error) {
	c.Lock()
	defer c.Unlock()

	found := false
	errs := []string{}
	var err error
	for _, e := range c.engines {
		for _, volume := range e.Volumes() {
			if volume.Name == name {
				if err := volume.Engine.RemoveVolume(name); err != nil {
					errs = append(errs, fmt.Sprintf("%s: %s", volume.Engine.Name, err.Error()))
					continue
				}
				found = true
			}
		}
	}
	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, "\n"))
	}
	return found, err
}

// Pull is exported
func (c *Cluster) Pull(name string, authConfig *dockerclient.AuthConfig, callback func(where, status string, err error)) {
	var wg sync.WaitGroup

	c.RLock()
	for _, e := range c.engines {
		wg.Add(1)

		go func(engine *cluster.Engine) {
			defer wg.Done()

			if callback != nil {
				callback(engine.Name, "", nil)
			}
			err := engine.Pull(name, authConfig)
			if callback != nil {
				if err != nil {
					callback(engine.Name, "", err)
				} else {
					callback(engine.Name, "downloaded", nil)
				}
			}
		}(e)
	}
	c.RUnlock()

	wg.Wait()
}

// Load image
func (c *Cluster) Load(imageReader io.Reader, callback func(where, status string, err error)) {
	var wg sync.WaitGroup

	c.RLock()
	pipeWriters := []*io.PipeWriter{}
	for _, e := range c.engines {
		wg.Add(1)

		pipeReader, pipeWriter := io.Pipe()
		pipeWriters = append(pipeWriters, pipeWriter)

		go func(reader *io.PipeReader, engine *cluster.Engine) {
			defer wg.Done()
			defer reader.Close()

			// call engine load image
			err := engine.Load(reader)
			if callback != nil {
				if err != nil {
					callback(engine.Name, "", err)
				}
			}
		}(pipeReader, e)
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
func (c *Cluster) Import(source string, repository string, tag string, imageReader io.Reader, callback func(what, status string, err error)) {
	var wg sync.WaitGroup
	c.RLock()
	pipeWriters := []*io.PipeWriter{}

	for _, e := range c.engines {
		wg.Add(1)

		pipeReader, pipeWriter := io.Pipe()
		pipeWriters = append(pipeWriters, pipeWriter)

		go func(reader *io.PipeReader, engine *cluster.Engine) {
			defer wg.Done()
			defer reader.Close()

			// call engine import
			err := engine.Import(source, repository, tag, reader)
			if callback != nil {
				if err != nil {
					callback(engine.Name, "", err)
				} else {
					callback(engine.Name, "Import success", nil)
				}
			}

		}(pipeReader, e)
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
	for _, e := range c.engines {
		out = append(out, e.Containers()...)
	}

	return out
}

func (c *Cluster) checkNameUniqueness(name string) bool {
	// Abort immediately if the name is empty.
	if len(name) == 0 {
		return true
	}

	c.RLock()
	defer c.RUnlock()
	for _, e := range c.engines {
		for _, c := range e.Containers() {
			for _, cname := range c.Names {
				if cname == name || cname == "/"+name {
					return false
				}
			}
		}
	}

	// check pending containers.
	for _, c := range c.pendingContainers {
		if c.Name == name {
			return false
		}
	}

	return true
}

// Container returns the container with IDOrName in the cluster
func (c *Cluster) Container(IDOrName string) *cluster.Container {
	// Abort immediately if the name is empty.
	if len(IDOrName) == 0 {
		return nil
	}

	c.RLock()
	defer c.RUnlock()

	return c.Containers().Get(IDOrName)
}

// Networks returns all the networks in the cluster.
func (c *Cluster) Networks() cluster.Networks {
	c.RLock()
	defer c.RUnlock()

	out := cluster.Networks{}
	for _, e := range c.engines {
		out = append(out, e.Networks()...)
	}

	return out
}

// Volumes returns all the volumes in the cluster.
func (c *Cluster) Volumes() []*cluster.Volume {
	c.RLock()
	defer c.RUnlock()

	out := []*cluster.Volume{}
	for _, e := range c.engines {
		out = append(out, e.Volumes()...)
	}

	return out
}

// Volume returns the volume name in the cluster
func (c *Cluster) Volume(name string) *cluster.Volume {
	// Abort immediately if the name is empty.
	if len(name) == 0 {
		return nil
	}

	c.RLock()
	defer c.RUnlock()

	for _, e := range c.engines {
		for _, v := range e.Volumes() {
			if v.Name == name {
				return v
			}
		}
	}
	return nil
}

// listNodes returns all the engines in the cluster.
func (c *Cluster) listNodes() []*node.Node {
	c.RLock()
	defer c.RUnlock()

	out := make([]*node.Node, 0, len(c.engines))
	for _, e := range c.engines {
		node := node.NewNode(e)
		for _, c := range c.pendingContainers {
			if c.Engine.ID == e.ID && node.Container(c.Config.SwarmID()) == nil {
				node.AddContainer(c.ToContainer())
			}
		}
		out = append(out, node)
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
	nodes, err := c.scheduler.SelectNodesForContainer(c.listNodes(), &cluster.ContainerConfig{})
	if err != nil {
		return nil, err
	}
	return c.engines[nodes[0].ID], nil
}

// RenameContainer rename a container
func (c *Cluster) RenameContainer(container *cluster.Container, newName string) error {
	c.RLock()
	defer c.RUnlock()

	// check new name whether available
	if !c.checkNameUniqueness(newName) {
		return fmt.Errorf("Conflict: The name %s is already assigned. You have to delete (or rename) that container to be able to assign %s to a container again.", newName, newName)
	}

	// call engine rename
	err := container.Engine.RenameContainer(container, newName)
	return err
}

// BuildImage build an image
func (c *Cluster) BuildImage(buildImage *dockerclient.BuildImage, out io.Writer) error {
	c.scheduler.Lock()

	// get an engine
	config := &cluster.ContainerConfig{dockerclient.ContainerConfig{
		CpuShares: buildImage.CpuShares,
		Memory:    buildImage.Memory,
	}}
	nodes, err := c.scheduler.SelectNodesForContainer(c.listNodes(), config)
	c.scheduler.Unlock()
	if err != nil {
		return err
	}
	n := nodes[0]

	reader, err := c.engines[n.ID].BuildImage(buildImage)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, reader); err != nil {
		return err
	}

	c.engines[n.ID].RefreshImages()
	return nil
}

// TagImage tag an image
func (c *Cluster) TagImage(IDOrName string, repo string, tag string, force bool) error {
	c.RLock()
	defer c.RUnlock()

	errs := []string{}
	var err error
	found := false
	for _, e := range c.engines {
		for _, image := range e.Images(true, nil) {
			if image.Match(IDOrName, true) {
				found = true
				err := image.Engine.TagImage(IDOrName, repo, tag, force)
				if err != nil {
					errs = append(errs, fmt.Sprintf("%s: %s", image.Engine.Name, err.Error()))
					continue
				}
			}
		}
	}
	if !found {
		return fmt.Errorf("No such image: %s", IDOrName)
	}
	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, "\n"))
	}

	return err
}
