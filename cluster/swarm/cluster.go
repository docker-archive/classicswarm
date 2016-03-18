package swarm

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/discovery"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/engine-api/types"
	"github.com/docker/go-units"
	"github.com/docker/swarm/cluster"
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
		Container: dockerclient.Container{
			Labels: p.Config.Labels,
		},
		Config: p.Config,
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

	eventHandlers     *cluster.EventHandlers
	engines           map[string]*cluster.Engine
	pendingEngines    map[string]*cluster.Engine
	scheduler         *scheduler.Scheduler
	discovery         discovery.Backend
	pendingContainers map[string]*pendingContainer

	overcommitRatio float64
	engineOpts      *cluster.EngineOpts
	createRetry     int64
	TLSConfig       *tls.Config
}

// NewCluster is exported
func NewCluster(scheduler *scheduler.Scheduler, TLSConfig *tls.Config, discovery discovery.Backend, options cluster.DriverOpts, engineOptions *cluster.EngineOpts) (cluster.Cluster, error) {
	log.WithFields(log.Fields{"name": "swarm"}).Debug("Initializing cluster")

	cluster := &Cluster{
		eventHandlers:     cluster.NewEventHandlers(),
		engines:           make(map[string]*cluster.Engine),
		pendingEngines:    make(map[string]*cluster.Engine),
		scheduler:         scheduler,
		TLSConfig:         TLSConfig,
		discovery:         discovery,
		pendingContainers: make(map[string]*pendingContainer),
		overcommitRatio:   0.05,
		engineOpts:        engineOptions,
		createRetry:       0,
	}

	if val, ok := options.Float("swarm.overcommit", ""); ok {
		if val <= float64(-1) {
			log.Fatalf("swarm.overcommit should be larger than -1, %f is invalid", val)
		} else if val < float64(0) {
			log.Warn("-1 < swarm.overcommit < 0 will make swarm take less resource than docker engine offers")
			cluster.overcommitRatio = val
		} else {
			cluster.overcommitRatio = val
		}
	}

	if val, ok := options.Int("swarm.createretry", ""); ok {
		if val < 0 {
			log.Fatalf("swarm.createretry can not be negative, %d is invalid", val)
		}
		cluster.createRetry = val
	}

	discoveryCh, errCh := cluster.discovery.Watch(nil)
	go cluster.monitorDiscovery(discoveryCh, errCh)
	go cluster.monitorPendingEngines()

	return cluster, nil
}

// Handle callbacks for the events
func (c *Cluster) Handle(e *cluster.Event) error {
	c.eventHandlers.Handle(e)
	return nil
}

// RegisterEventHandler registers an event handler.
func (c *Cluster) RegisterEventHandler(h cluster.EventHandler) error {
	return c.eventHandlers.RegisterEventHandler(h)
}

// UnregisterEventHandler unregisters a previously registered event handler.
func (c *Cluster) UnregisterEventHandler(h cluster.EventHandler) {
	c.eventHandlers.UnregisterEventHandler(h)
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

// StartContainer starts a container
func (c *Cluster) StartContainer(container *cluster.Container, hostConfig *dockerclient.HostConfig) error {
	return container.Engine.StartContainer(container.Id, hostConfig)
}

// CreateContainer aka schedule a brand new container into the cluster.
func (c *Cluster) CreateContainer(config *cluster.ContainerConfig, name string, authConfig *dockerclient.AuthConfig) (*cluster.Container, error) {
	container, err := c.createContainer(config, name, false, authConfig)

	if err != nil {
		var retries int64
		//  fails with image not found, then try to reschedule with image affinity
		bImageNotFoundError, _ := regexp.MatchString(`image \S* not found`, err.Error())
		if bImageNotFoundError && !config.HaveNodeConstraint() {
			// Check if the image exists in the cluster
			// If exists, retry with a image affinity
			if c.Image(config.Image) != nil {
				container, err = c.createContainer(config, name, true, authConfig)
				retries++
			}
		}

		for ; retries < c.createRetry && err != nil; retries++ {
			log.WithFields(log.Fields{"Name": "Swarm"}).Warnf("Failed to create container: %s, retrying", err)
			container, err = c.createContainer(config, name, false, authConfig)
		}
	}
	return container, err
}

func (c *Cluster) createContainer(config *cluster.ContainerConfig, name string, withImageAffinity bool, authConfig *dockerclient.AuthConfig) (*cluster.Container, error) {
	c.scheduler.Lock()

	// Ensure the name is available
	if !c.checkNameUniqueness(name) {
		c.scheduler.Unlock()
		return nil, fmt.Errorf("Conflict: The name %s is already assigned. You have to delete (or rename) that container to be able to assign %s to a container again.", name, name)
	}

	swarmID := config.SwarmID()
	if swarmID == "" {
		// Associate a Swarm ID to the container we are creating.
		swarmID = c.generateUniqueID()
		config.SetSwarmID(swarmID)
	}

	if network := c.Networks().Get(config.HostConfig.NetworkMode); network != nil && network.Scope == "local" {
		if !config.HaveNodeConstraint() {
			config.AddConstraint("node==~" + network.Engine.Name)
		}
		config.HostConfig.NetworkMode = network.Name
	}

	if withImageAffinity {
		config.AddAffinity("image==" + config.Image)
	}

	nodes, err := c.scheduler.SelectNodesForContainer(c.listNodes(), config)

	if withImageAffinity {
		config.RemoveAffinity("image==" + config.Image)
	}

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

	container, err := engine.Create(config, name, true, authConfig)

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
	if err == nil && network.Scope == "global" {
		for _, engine := range c.engines {
			engine.DeleteNetwork(network)
		}
	}
	return err
}

func (c *Cluster) getEngineByAddr(addr string) *cluster.Engine {
	c.RLock()
	defer c.RUnlock()

	if engine, ok := c.pendingEngines[addr]; ok {
		return engine
	}
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

	engine := cluster.NewEngine(addr, c.overcommitRatio, c.engineOpts)
	if err := engine.RegisterEventHandler(c); err != nil {
		log.Error(err)
	}
	// Add it to pending engine map, indexed by address. This will prevent
	// duplicates from entering
	c.Lock()
	c.pendingEngines[addr] = engine
	c.Unlock()

	// validatePendingEngine will start a thread to validate the engine.
	// If the engine is reachable and valid, it'll be monitored and updated in a loop.
	// If engine is not reachable, pending engines will be examined once in a while
	go c.validatePendingEngine(engine)

	return true
}

// validatePendingEngine connects to the engine,
func (c *Cluster) validatePendingEngine(engine *cluster.Engine) bool {
	// Attempt a connection to the engine. Since this is slow, don't get a hold
	// of the lock yet.
	if err := engine.Connect(c.TLSConfig); err != nil {
		log.WithFields(log.Fields{"Addr": engine.Addr}).Debugf("Failed to validate pending node: %s", err)
		return false
	}

	// The following is critical and fast. Grab a lock.
	c.Lock()
	defer c.Unlock()

	// Only validate engines from pendingEngines list
	if _, exists := c.pendingEngines[engine.Addr]; !exists {
		return false
	}

	// Make sure the engine ID is unique.
	if old, exists := c.engines[engine.ID]; exists {
		if old.Addr != engine.Addr {
			log.Errorf("ID duplicated. %s shared by %s and %s", engine.ID, old.Addr, engine.Addr)
			// Keep this engine in pendingEngines table and show its error.
			// If it's ID duplication from VM clone, user see this message and can fix it.
			// If the engine is rebooted and get new IP from DHCP, previous address will be removed
			// from discovery after a while.
			// In both cases, retry may fix the problem.
			engine.HandleIDConflict(old.Addr)
		} else {
			log.Debugf("node %q (name: %q) with address %q is already registered", engine.ID, engine.Name, engine.Addr)
			engine.Disconnect()
			// Remove it from pendingEngines table
			delete(c.pendingEngines, engine.Addr)
		}
		return false
	}

	// Engine validated, move from pendingEngines table to engines table
	delete(c.pendingEngines, engine.Addr)
	// set engine state to healthy, and start refresh loop
	engine.ValidationComplete()
	c.engines[engine.ID] = engine

	log.Infof("Registered Engine %s at %s", engine.Name, engine.Addr)
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
	// it could be in pendingEngines or engines
	if _, ok := c.pendingEngines[addr]; ok {
		delete(c.pendingEngines, addr)
	} else {
		delete(c.engines, engine.ID)
	}
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

			for _, entry := range added {
				c.addEngine(entry.String())
			}
		case err := <-errCh:
			log.Errorf("Discovery error: %v", err)
		}
	}
}

// monitorPendingEngines checks if some previous unreachable/invalid engines have been fixed
func (c *Cluster) monitorPendingEngines() {
	const minimumValidationInterval time.Duration = 10 * time.Second
	for {
		// Don't need to do it frequently
		time.Sleep(minimumValidationInterval)
		// Get the list of pendingEngines
		c.RLock()
		pEngines := make([]*cluster.Engine, 0, len(c.pendingEngines))
		for _, e := range c.pendingEngines {
			pEngines = append(pEngines, e)
		}
		c.RUnlock()
		for _, e := range pEngines {
			if e.TimeToValidate() {
				go c.validatePendingEngine(e)
			}
		}
	}
}

// Images returns all the images in the cluster.
func (c *Cluster) Images() cluster.Images {
	c.RLock()
	defer c.RUnlock()

	out := []*cluster.Image{}
	for _, e := range c.engines {
		out = append(out, e.Images()...)
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
func (c *Cluster) RemoveImages(name string, force bool) ([]types.ImageDelete, error) {
	c.Lock()
	defer c.Unlock()

	out := []types.ImageDelete{}
	errs := []string{}
	var err error
	for _, e := range c.engines {
		for _, image := range e.Images() {
			if image.Match(name, true) {
				content, err := image.Engine.RemoveImage(name, force)
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

func (c *Cluster) refreshVolumes() {
	var wg sync.WaitGroup
	for _, e := range c.engines {
		wg.Add(1)
		go func(e *cluster.Engine) {
			e.RefreshVolumes()
			wg.Done()
		}(e)
	}
	wg.Wait()
}

// CreateNetwork creates a network in the cluster
func (c *Cluster) CreateNetwork(request *types.NetworkCreate) (response *types.NetworkCreateResponse, err error) {
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
		if err == nil {
			if network := c.engines[nodes[0].ID].Networks().Get(resp.ID); network != nil && network.Scope == "global" {
				for id, engine := range c.engines {
					if id != nodes[0].ID {
						engine.AddNetwork(network)
					}
				}
			}
		}
		return resp, err
	}
	return nil, nil
}

// CreateVolume creates a volume in the cluster
func (c *Cluster) CreateVolume(request *types.VolumeCreateRequest) (*cluster.Volume, error) {
	var (
		wg     sync.WaitGroup
		volume *cluster.Volume
		err    error
		parts  = strings.SplitN(request.Name, "/", 2)
		node   = ""
	)

	if request.Name == "" {
		request.Name = stringid.GenerateRandomID()
	} else if len(parts) == 2 {
		node = parts[0]
		request.Name = parts[1]
	}
	if node == "" {
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
	} else {
		config := cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:node==" + parts[0]}})
		nodes, err := c.scheduler.SelectNodesForContainer(c.listNodes(), config)
		if err != nil {
			return nil, err
		}
		if nodes != nil {
			v, er := c.engines[nodes[0].ID].CreateVolume(request)
			if v != nil {
				volume = v
				err = nil
			}
			if er != nil && volume == nil {
				err = er
			}
		}
	}

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
		if volume := e.Volumes().Get(name); volume != nil {
			if err := volume.Engine.RemoveVolume(volume.Name); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %s", volume.Engine.Name, err.Error()))
				continue
			}
			found = true
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
func (c *Cluster) Volumes() cluster.Volumes {
	c.RLock()
	defer c.RUnlock()

	out := cluster.Volumes{}
	for _, e := range c.engines {
		out = append(out, e.Volumes()...)
	}

	return out
}

// listNodes returns all validated engines in the cluster, excluding pendingEngines.
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
// This is for reporting, not scheduling, hence pendingEngines are included.
func (c *Cluster) listEngines() []*cluster.Engine {
	c.RLock()
	defer c.RUnlock()

	out := make([]*cluster.Engine, 0, len(c.engines)+len(c.pendingEngines))
	for _, n := range c.engines {
		out = append(out, n)
	}
	for _, n := range c.pendingEngines {
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
func (c *Cluster) TotalCpus() int {
	var totalCpus int
	for _, engine := range c.engines {
		totalCpus += engine.TotalCpus()
	}
	return totalCpus
}

// Info returns some info about the cluster, like nb or containers / images
func (c *Cluster) Info() [][2]string {
	info := [][2]string{
		{"Strategy", c.scheduler.Strategy()},
		{"Filters", c.scheduler.Filters()},
		{"Nodes", fmt.Sprintf("%d", len(c.engines)+len(c.pendingEngines))},
	}

	engines := c.listEngines()
	sort.Sort(cluster.EngineSorter(engines))

	for _, engine := range engines {
		engineName := "(unknown)"
		if engine.Name != "" {
			engineName = engine.Name
		}
		info = append(info, [2]string{" " + engineName, engine.Addr})
		info = append(info, [2]string{"  └ Status", engine.Status()})
		info = append(info, [2]string{"  └ Containers", fmt.Sprintf("%d", len(engine.Containers()))})
		info = append(info, [2]string{"  └ Reserved CPUs", fmt.Sprintf("%d / %d", engine.UsedCpus(), engine.TotalCpus())})
		info = append(info, [2]string{"  └ Reserved Memory", fmt.Sprintf("%s / %s", units.BytesSize(float64(engine.UsedMemory())), units.BytesSize(float64(engine.TotalMemory())))})
		labels := make([]string, 0, len(engine.Labels))
		for k, v := range engine.Labels {
			labels = append(labels, k+"="+v)
		}
		sort.Strings(labels)
		info = append(info, [2]string{"  └ Labels", fmt.Sprintf("%s", strings.Join(labels, ", "))})
		errMsg := engine.ErrMsg()
		if len(errMsg) == 0 {
			errMsg = "(none)"
		}
		info = append(info, [2]string{"  └ Error", errMsg})
		info = append(info, [2]string{"  └ UpdatedAt", engine.UpdatedAt().UTC().Format(time.RFC3339)})
		info = append(info, [2]string{"  └ ServerVersion", engine.Version})
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
func (c *Cluster) BuildImage(buildImage *types.ImageBuildOptions, out io.Writer) error {
	c.scheduler.Lock()

	// get an engine
	config := cluster.BuildContainerConfig(dockerclient.ContainerConfig{
		CpuShares: buildImage.CPUShares,
		Memory:    buildImage.Memory,
		Env:       convertMapToKVStrings(buildImage.BuildArgs),
	})
	buildImage.BuildArgs = convertKVStringsToMap(config.Env)
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
		for _, image := range e.Images() {
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
