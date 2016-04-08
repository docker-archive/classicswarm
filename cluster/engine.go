package cluster

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/version"
	engineapi "github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	engineapinop "github.com/docker/swarm/api/nopclient"
	"github.com/samalba/dockerclient"
	"github.com/samalba/dockerclient/nopclient"
)

const (
	// Timeout for requests sent out to the engine.
	requestTimeout = 10 * time.Second

	// Minimum docker engine version supported by swarm.
	minSupportedVersion = version.Version("1.6.0")
)

type engineState int

const (
	// pending means an engine added to cluster has not been validated
	statePending engineState = iota
	// unhealthy means an engine is unreachable
	stateUnhealthy
	// healthy means an engine is reachable
	stateHealthy
	// disconnected means engine is removed from discovery
	stateDisconnected

	// TODO: add maintenance state. Proposal #1486
	// maintenance means an engine is under maintenance.
	// There is no action to migrate a node into maintenance state yet.
	//stateMaintenance
)

var stateText = map[engineState]string{
	statePending:      "Pending",
	stateUnhealthy:    "Unhealthy",
	stateHealthy:      "Healthy",
	stateDisconnected: "Disconnected",
	//stateMaintenance: "Maintenance",
}

// delayer offers a simple API to random delay within a given time range.
type delayer struct {
	rangeMin time.Duration
	rangeMax time.Duration

	r *rand.Rand
	l sync.Mutex
}

func newDelayer(rangeMin, rangeMax time.Duration) *delayer {
	return &delayer{
		rangeMin: rangeMin,
		rangeMax: rangeMax,
		r:        rand.New(rand.NewSource(time.Now().UTC().UnixNano())),
	}
}

// Wait returns timeout event after fixed + randomized time duration
func (d *delayer) Wait(backoffFactor int) <-chan time.Time {
	d.l.Lock()
	defer d.l.Unlock()

	waitPeriod := int64(d.rangeMin) * int64(1+backoffFactor)
	if delta := int64(d.rangeMax) - int64(d.rangeMin); delta > 0 {
		// Int63n panics if the parameter is 0
		waitPeriod += d.r.Int63n(delta)
	}
	return time.After(time.Duration(waitPeriod))
}

// EngineOpts represents the options for an engine
type EngineOpts struct {
	RefreshMinInterval time.Duration
	RefreshMaxInterval time.Duration
	FailureRetry       int
}

// Engine represents a docker engine
type Engine struct {
	sync.RWMutex

	ID      string
	IP      string
	Addr    string
	Name    string
	Cpus    int
	Memory  int64
	Labels  map[string]string
	Version string

	stopCh          chan struct{}
	refreshDelayer  *delayer
	containers      map[string]*Container
	images          []*Image
	networks        map[string]*Network
	volumes         map[string]*Volume
	client          dockerclient.Client
	apiClient       engineapi.APIClient
	eventHandler    EventHandler
	state           engineState
	lastError       string
	updatedAt       time.Time
	failureCount    int
	overcommitRatio int64
	opts            *EngineOpts
}

// NewEngine is exported
func NewEngine(addr string, overcommitRatio float64, opts *EngineOpts) *Engine {
	e := &Engine{
		Addr:            addr,
		client:          nopclient.NewNopClient(),
		apiClient:       engineapinop.NewNopClient(),
		refreshDelayer:  newDelayer(opts.RefreshMinInterval, opts.RefreshMaxInterval),
		Labels:          make(map[string]string),
		stopCh:          make(chan struct{}),
		containers:      make(map[string]*Container),
		networks:        make(map[string]*Network),
		volumes:         make(map[string]*Volume),
		state:           statePending,
		updatedAt:       time.Now(),
		overcommitRatio: int64(overcommitRatio * 100),
		opts:            opts,
	}
	return e
}

// HTTPClientAndScheme returns the underlying HTTPClient and the scheme used by the engine
func (e *Engine) HTTPClientAndScheme() (*http.Client, string, error) {
	if dc, ok := e.client.(*dockerclient.DockerClient); ok {
		return dc.HTTPClient, dc.URL.Scheme, nil
	}
	return nil, "", fmt.Errorf("Possibly lost connection to Engine (name: %s, ID: %s) ", e.Name, e.ID)
}

// Connect will initialize a connection to the Docker daemon running on the
// host, gather machine specs (memory, cpu, ...) and monitor state changes.
func (e *Engine) Connect(config *tls.Config) error {
	host, _, err := net.SplitHostPort(e.Addr)
	if err != nil {
		return err
	}

	addr, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return err
	}
	e.IP = addr.IP.String()

	c, err := dockerclient.NewDockerClientTimeout("tcp://"+e.Addr, config, time.Duration(requestTimeout), setTCPUserTimeout)
	if err != nil {
		return err
	}
	// Use HTTP Client used by dockerclient to create engine-api client
	apiClient, err := engineapi.NewClient("tcp://"+e.Addr, "", c.HTTPClient, nil)
	if err != nil {
		return err
	}

	return e.ConnectWithClient(c, apiClient)
}

// StartMonitorEvents monitors events from the engine
func (e *Engine) StartMonitorEvents() {
	log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).Debug("Start monitoring events")
	ec := make(chan error)
	e.client.StartMonitorEvents(e.handler, ec)

	go func() {
		if err := <-ec; err != nil {
			log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).Errorf("Error monitoring events: %s.", err)
			if !strings.Contains(err.Error(), "EOF") {
				// failing node reconnect should use back-off strategy
				<-e.refreshDelayer.Wait(e.getFailureCount())
			}
			log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).Errorf("Restart event monitoring.")
			e.StartMonitorEvents()
		}
		close(ec)
	}()
}

// ConnectWithClient is exported
func (e *Engine) ConnectWithClient(client dockerclient.Client, apiClient engineapi.APIClient) error {
	e.client = client
	e.apiClient = apiClient

	// Fetch the engine labels.
	if err := e.updateSpecs(); err != nil {
		return err
	}

	e.StartMonitorEvents()

	// Force a state update before returning.
	if err := e.RefreshContainers(true); err != nil {
		return err
	}

	if err := e.RefreshImages(); err != nil {
		return err
	}

	// Do not check error as older daemon does't support this call.
	e.RefreshVolumes()
	e.RefreshNetworks()

	e.emitEvent("engine_connect")

	return nil
}

// Disconnect will stop all monitoring of the engine.
// The Engine object cannot be further used without reconnecting it first.
func (e *Engine) Disconnect() {
	e.Lock()
	defer e.Unlock()
	// Resource clean up should be done only once
	if e.state == stateDisconnected {
		return
	}

	// close the chan
	close(e.stopCh)
	e.client.StopAllMonitorEvents()
	// close idle connections
	if dc, ok := e.client.(*dockerclient.DockerClient); ok {
		closeIdleConnections(dc.HTTPClient)
	}
	e.client = nopclient.NewNopClient()
	e.apiClient = engineapinop.NewNopClient()
	e.state = stateDisconnected
	e.emitEvent("engine_disconnect")
}

func closeIdleConnections(client *http.Client) {
	if tr, ok := client.Transport.(*http.Transport); ok {
		tr.CloseIdleConnections()
	}
}

// isConnected returns true if the engine is connected to a remote docker API
// note that it's not the same as stateDisconnected. Engine isConnected is also true
// when it is first created but not yet 'Connect' to a remote docker API.
func (e *Engine) isConnected() bool {
	_, ok := e.client.(*nopclient.NopClient)
	_, okAPIClient := e.apiClient.(*engineapinop.NopClient)
	return (!ok && !okAPIClient)
}

// IsHealthy returns true if the engine is healthy
func (e *Engine) IsHealthy() bool {
	e.RLock()
	defer e.RUnlock()
	return e.state == stateHealthy
}

// HealthIndicator returns degree of healthiness between 0 and 100.
// 0 means node is not healthy (unhealthy, pending), 100 means last connectivity was successful
// other values indicate recent failures but haven't moved engine out of healthy state
func (e *Engine) HealthIndicator() int64 {
	e.RLock()
	defer e.RUnlock()
	if e.state != stateHealthy || e.failureCount >= e.opts.FailureRetry {
		return 0
	}
	return int64(100 - e.failureCount*100/e.opts.FailureRetry)
}

// setState sets engine state
func (e *Engine) setState(state engineState) {
	e.Lock()
	defer e.Unlock()
	e.state = state
}

// TimeToValidate returns true if a pending node is up for validation
func (e *Engine) TimeToValidate() bool {
	const validationLimit time.Duration = 4 * time.Hour
	const minFailureBackoff time.Duration = 30 * time.Second
	e.Lock()
	defer e.Unlock()
	if e.state != statePending {
		return false
	}
	sinceLastUpdate := time.Since(e.updatedAt)
	// Increase check interval for a pending engine according to failureCount and cap it at a limit
	// it's exponential backoff = 2 ^ failureCount + minFailureBackoff. A minimum backoff is
	// needed because e.failureCount could be 0 at first join, or the engine has a duplicate ID
	if sinceLastUpdate > validationLimit ||
		sinceLastUpdate > (1<<uint(e.failureCount))*time.Second+minFailureBackoff {
		return true
	}
	return false
}

// ValidationComplete transitions engine state from statePending to stateHealthy
func (e *Engine) ValidationComplete() {
	e.Lock()
	defer e.Unlock()
	if e.state != statePending {
		return
	}
	e.state = stateHealthy
	e.failureCount = 0
	go e.refreshLoop()
}

// setErrMsg sets error message for the engine
func (e *Engine) setErrMsg(errMsg string) {
	e.Lock()
	defer e.Unlock()
	e.lastError = strings.TrimSpace(errMsg)
	e.updatedAt = time.Now()
}

// ErrMsg returns error message for the engine
func (e *Engine) ErrMsg() string {
	e.RLock()
	defer e.RUnlock()
	return e.lastError
}

// HandleIDConflict handles ID duplicate with existing engine
func (e *Engine) HandleIDConflict(otherAddr string) {
	e.setErrMsg(fmt.Sprintf("ID duplicated. %s shared by this node %s and another node %s", e.ID, e.Addr, otherAddr))
}

// Status returns the health status of the Engine: Healthy or Unhealthy
func (e *Engine) Status() string {
	e.RLock()
	defer e.RUnlock()
	return stateText[e.state]
}

// incFailureCount increases engine's failure count, and sets engine as unhealthy if threshold is crossed
func (e *Engine) incFailureCount() {
	e.Lock()
	defer e.Unlock()
	e.failureCount++
	if e.state == stateHealthy && e.failureCount >= e.opts.FailureRetry {
		e.state = stateUnhealthy
		log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).Errorf("Flagging engine as unhealthy. Connect failed %d times", e.failureCount)
		e.emitEvent("engine_disconnect")
	}
}

// getFailureCount returns a copy on the getFailureCount, thread-safe
func (e *Engine) getFailureCount() int {
	e.RLock()
	defer e.RUnlock()
	return e.failureCount
}

// UpdatedAt returns the previous updatedAt time
func (e *Engine) UpdatedAt() time.Time {
	e.RLock()
	defer e.RUnlock()
	return e.updatedAt
}

func (e *Engine) resetFailureCount() {
	e.Lock()
	defer e.Unlock()
	e.failureCount = 0
}

// CheckConnectionErr checks error from client response and adjusts engine healthy indicators
func (e *Engine) CheckConnectionErr(err error) {
	if err == nil {
		e.setErrMsg("")
		// If current state is unhealthy, change it to healthy
		if e.state == stateUnhealthy {
			log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).Infof("Engine came back to life after %d retries. Hooray!", e.getFailureCount())
			e.emitEvent("engine_reconnect")
			e.setState(stateHealthy)
		}
		e.resetFailureCount()
		return
	}

	// dockerclient defines ErrConnectionRefused error. but if http client is from swarm, it's not using
	// dockerclient. We need string matching for these cases. Remove the first character to deal with
	// case sensitive issue.
	// engine-api returns ErrConnectionFailed error, so we check for that as long as dockerclient exists
	if err == dockerclient.ErrConnectionRefused ||
		err == engineapi.ErrConnectionFailed ||
		strings.Contains(err.Error(), "onnection refused") ||
		strings.Contains(err.Error(), "annot connect to the docker engine endpoint") {
		// each connection refused instance may increase failure count so
		// engine can fail fast. Short engine freeze or network failure may result
		// in engine marked as unhealthy. If this causes unnecessary failure, engine
		// can track last error time. Only increase failure count if last error is
		// not too recent, e.g., last error is at least 1 seconds ago.
		e.incFailureCount()
		// update engine error message
		e.setErrMsg(err.Error())
		return
	}
	// other errors may be ambiguous.
}

// Update API Version in apiClient
func (e *Engine) updateClientVersionFromServer(serverVersion string) {
	// v will be >= 1.6, since this is checked earlier
	v := version.Version(serverVersion)
	switch {
	case v.LessThan(version.Version("1.7")):
		e.apiClient.UpdateClientVersion("1.18")
	case v.LessThan(version.Version("1.8")):
		e.apiClient.UpdateClientVersion("1.19")
	case v.LessThan(version.Version("1.9")):
		e.apiClient.UpdateClientVersion("1.20")
	case v.LessThan(version.Version("1.10")):
		e.apiClient.UpdateClientVersion("1.21")
	case v.LessThan(version.Version("1.11")):
		e.apiClient.UpdateClientVersion("1.22")
	default:
		e.apiClient.UpdateClientVersion("1.23")
	}
}

// Gather engine specs (CPU, memory, constraints, ...).
func (e *Engine) updateSpecs() error {
	info, err := e.apiClient.Info(context.TODO())
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	if info.NCPU == 0 || info.MemTotal == 0 {
		return fmt.Errorf("cannot get resources for this engine, make sure %s is a Docker Engine, not a Swarm manager", e.Addr)
	}

	v, err := e.apiClient.ServerVersion(context.TODO())
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	engineVersion := version.Version(v.Version)

	// Older versions of Docker don't expose the ID field, Labels and are not supported
	// by Swarm.  Catch the error ASAP and refuse to connect.
	if engineVersion.LessThan(minSupportedVersion) {
		err = fmt.Errorf("engine %s is running an unsupported version of Docker Engine. Please upgrade to at least %s", e.Addr, minSupportedVersion)
		e.CheckConnectionErr(err)
		return err
	}
	// update server version
	e.Version = v.Version
	// update client version. engine-api handles backward compatibility where needed
	e.updateClientVersionFromServer(v.Version)

	e.Lock()
	defer e.Unlock()
	// Swarm/docker identifies engine by ID. Updating ID but not updating cluster
	// index will put the cluster into inconsistent state. If this happens, the
	// engine should be put to pending state for re-validation.
	if e.ID == "" {
		e.ID = info.ID
	} else if e.ID != info.ID {
		e.state = statePending
		message := fmt.Sprintf("Engine (ID: %s, Addr: %s) shows up with another ID:%s. Please remove it from cluster, it can be added back.", e.ID, e.Addr, info.ID)
		e.lastError = message
		return fmt.Errorf(message)
	}
	e.Name = info.Name
	e.Cpus = info.NCPU
	e.Memory = info.MemTotal
	e.Labels = map[string]string{
		"storagedriver":   info.Driver,
		"executiondriver": info.ExecutionDriver,
		"kernelversion":   info.KernelVersion,
		"operatingsystem": info.OperatingSystem,
	}
	for _, label := range info.Labels {
		kv := strings.SplitN(label, "=", 2)
		if len(kv) != 2 {
			message := fmt.Sprintf("Engine (ID: %s, Addr: %s) contains an invalid label (%s) not formatted as \"key=value\".", e.ID, e.Addr, label)
			return fmt.Errorf(message)
		}

		// If an engine managed by Swarm contains a label with key "node",
		// such as node=node1
		// `docker run -e constraint:node==node1 -d nginx` will not work,
		// since "node" in constraint will match node.Name instead of label.
		// Log warn message in this case.
		if kv[0] == "node" {
			log.Warnf("Engine (ID: %s, Addr: %s) containers a label (%s) with key of \"node\" which cannot be used in Swarm.", e.ID, e.Addr, label)
		}

		e.Labels[kv[0]] = kv[1]
	}
	return nil
}

// RemoveImage deletes an image from the engine.
func (e *Engine) RemoveImage(name string, force bool) ([]types.ImageDelete, error) {
	rmOpts := types.ImageRemoveOptions{name, force, true}
	dels, err := e.apiClient.ImageRemove(context.TODO(), rmOpts)
	e.CheckConnectionErr(err)
	e.RefreshImages()
	return dels, err
}

// RemoveNetwork removes a network from the engine.
func (e *Engine) RemoveNetwork(network *Network) error {
	err := e.apiClient.NetworkRemove(context.TODO(), network.ID)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	// Remove the container from the state. Eventually, the state refresh loop
	// will rewrite this.
	e.DeleteNetwork(network)
	return nil
}

// DeleteNetwork deletes a network from the internal engine state.
func (e *Engine) DeleteNetwork(network *Network) {
	e.Lock()
	delete(e.networks, network.ID)
	e.Unlock()
}

// AddNetwork adds a network to the internal engine state.
func (e *Engine) AddNetwork(network *Network) {
	e.Lock()
	e.networks[network.ID] = &Network{
		NetworkResource: network.NetworkResource,
		Engine:          e,
	}
	e.Unlock()
}

// RemoveVolume deletes a volume from the engine.
func (e *Engine) RemoveVolume(name string) error {
	err := e.apiClient.VolumeRemove(context.TODO(), name)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	// Remove the container from the state. Eventually, the state refresh loop
	// will rewrite this.
	e.Lock()
	defer e.Unlock()
	delete(e.volumes, name)

	return nil
}

// RefreshImages refreshes the list of images on the engine.
func (e *Engine) RefreshImages() error {
	imgLstOpts := types.ImageListOptions{All: true}
	images, err := e.apiClient.ImageList(context.TODO(), imgLstOpts)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}
	e.Lock()
	e.images = nil
	for _, image := range images {
		e.images = append(e.images, &Image{Image: image, Engine: e})
	}
	e.Unlock()
	return nil
}

// RefreshNetworks refreshes the list of networks on the engine.
func (e *Engine) RefreshNetworks() error {
	netLsOpts := types.NetworkListOptions{filters.NewArgs()}
	networks, err := e.apiClient.NetworkList(context.TODO(), netLsOpts)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}
	e.Lock()
	e.networks = make(map[string]*Network)
	for _, network := range networks {
		e.networks[network.ID] = &Network{NetworkResource: network, Engine: e}
	}
	e.Unlock()
	return nil
}

// RefreshVolumes refreshes the list of volumes on the engine.
func (e *Engine) RefreshVolumes() error {
	volumesLsRsp, err := e.apiClient.VolumeList(context.TODO(), filters.NewArgs())
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}
	e.Lock()
	e.volumes = make(map[string]*Volume)
	for _, volume := range volumesLsRsp.Volumes {
		e.volumes[volume.Name] = &Volume{Volume: *volume, Engine: e}
	}
	e.Unlock()
	return nil
}

// RefreshContainers will refresh the list and status of containers running on the engine. If `full` is
// true, each container will be inspected.
// FIXME: unexport this method after mesos scheduler stops using it directly
func (e *Engine) RefreshContainers(full bool) error {
	containers, err := e.client.ListContainers(true, false, "")
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	merged := make(map[string]*Container)
	for _, c := range containers {
		mergedUpdate, err := e.updateContainer(c, merged, full)
		if err != nil {
			log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).Errorf("Unable to update state of container %q: %v", c.Id, err)
		} else {
			merged = mergedUpdate
		}
	}

	e.Lock()
	defer e.Unlock()
	e.containers = merged

	return nil
}

// Refresh the status of a container running on the engine. If `full` is true,
// the container will be inspected.
func (e *Engine) refreshContainer(ID string, full bool) (*Container, error) {
	containers, err := e.client.ListContainers(true, false, fmt.Sprintf("{%q:[%q]}", "id", ID))
	e.CheckConnectionErr(err)
	if err != nil {
		return nil, err
	}

	if len(containers) > 1 {
		// We expect one container, if we get more than one, trigger a full refresh.
		err = e.RefreshContainers(full)
		return nil, err
	}

	if len(containers) == 0 {
		// The container doesn't exist on the engine, remove it.
		e.Lock()
		delete(e.containers, ID)
		e.Unlock()

		return nil, nil
	}

	_, err = e.updateContainer(containers[0], e.containers, full)
	e.RLock()
	container := e.containers[containers[0].Id]
	e.RUnlock()
	return container, err
}

func (e *Engine) updateContainer(c dockerclient.Container, containers map[string]*Container, full bool) (map[string]*Container, error) {
	var container *Container

	e.RLock()
	if current, exists := e.containers[c.Id]; exists {
		// The container is already known.
		container = current
		// Restarting is a transit state. Unfortunately Docker doesn't always emit
		// events when it gets out of Restarting state. Force an inspect to update.
		if container.Info.State != nil && container.Info.State.Restarting {
			full = true
		}
	} else {
		// This is a brand new container. We need to do a full refresh.
		container = &Container{
			Engine: e,
		}
		full = true
	}
	// Release the lock here as the next step is slow.
	// Trade-off: If updateContainer() is called concurrently for the same
	// container, we will end up doing a full refresh twice and the original
	// container (containers[container.Id]) will get replaced.
	e.RUnlock()

	// Update ContainerInfo.
	if full {
		info, err := e.client.InspectContainer(c.Id)
		e.CheckConnectionErr(err)
		if err != nil {
			return nil, err
		}
		// Convert the ContainerConfig from inspect into our own
		// cluster.ContainerConfig.
		if info.HostConfig != nil {
			info.Config.HostConfig = *info.HostConfig
		}
		container.Config = BuildContainerConfig(*info.Config)

		// FIXME remove "duplicate" lines and move this to cluster/config.go
		container.Config.CpuShares = container.Config.CpuShares * int64(e.Cpus) / 1024.0
		container.Config.HostConfig.CpuShares = container.Config.CpuShares

		// Save the entire inspect back into the container.
		container.Info = *info
	}

	// Update its internal state.
	e.Lock()
	container.Container = c
	containers[container.Id] = container
	e.Unlock()

	return containers, nil
}

// refreshLoop periodically triggers engine refresh.
func (e *Engine) refreshLoop() {
	const maxBackoffFactor int = 1000
	// engine can hot-plug CPU/Mem or update labels. but there is no events
	// from engine to trigger spec update.
	// add an update interval and refresh spec for healthy nodes.
	const specUpdateInterval = 5 * time.Minute
	lastSpecUpdatedAt := time.Now()

	for {
		var err error

		// Engines keep failing should backoff
		// e.failureCount and e.opts.FailureRetry are type of int
		backoffFactor := e.getFailureCount() - e.opts.FailureRetry
		if backoffFactor < 0 {
			backoffFactor = 0
		} else if backoffFactor > maxBackoffFactor {
			backoffFactor = maxBackoffFactor
		}
		// Wait for the delayer or quit if we get stopped.
		select {
		case <-e.refreshDelayer.Wait(backoffFactor):
		case <-e.stopCh:
			return
		}

		healthy := e.IsHealthy()
		if !healthy || time.Since(lastSpecUpdatedAt) > specUpdateInterval {
			if err = e.updateSpecs(); err != nil {
				log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).Errorf("Update engine specs failed: %v", err)
				continue
			}
			lastSpecUpdatedAt = time.Now()
		}

		if !healthy {
			e.client.StopAllMonitorEvents()
			e.StartMonitorEvents()
		}

		err = e.RefreshContainers(false)
		if err == nil {
			// Do not check error as older daemon doesn't support this call
			e.RefreshVolumes()
			e.RefreshNetworks()
			e.RefreshImages()
			log.WithFields(log.Fields{"id": e.ID, "name": e.Name}).Debugf("Engine update succeeded")
		} else {
			log.WithFields(log.Fields{"id": e.ID, "name": e.Name}).Debugf("Engine refresh failed")
		}
	}
}

func (e *Engine) emitEvent(event string) {
	// If there is no event handler registered, abort right now.
	if e.eventHandler == nil {
		return
	}
	ev := &Event{
		Event: dockerclient.Event{
			Status: event,
			From:   "swarm",
			Type:   "swarm",
			Action: event,
			Actor: dockerclient.Actor{
				Attributes: make(map[string]string),
			},
			Time:     time.Now().Unix(),
			TimeNano: time.Now().UnixNano(),
		},
		Engine: e,
	}
	e.eventHandler.Handle(ev)
}

// UsedMemory returns the sum of memory reserved by containers.
func (e *Engine) UsedMemory() int64 {
	var r int64
	e.RLock()
	for _, c := range e.containers {
		r += c.Config.Memory
	}
	e.RUnlock()
	return r
}

// UsedCpus returns the sum of CPUs reserved by containers.
func (e *Engine) UsedCpus() int64 {
	var r int64
	e.RLock()
	for _, c := range e.containers {
		r += c.Config.CpuShares
	}
	e.RUnlock()
	return r
}

// TotalMemory returns the total memory + overcommit
func (e *Engine) TotalMemory() int64 {
	return e.Memory + (e.Memory * e.overcommitRatio / 100)
}

// TotalCpus returns the total cpus + overcommit
func (e *Engine) TotalCpus() int {
	return e.Cpus + (e.Cpus * int(e.overcommitRatio) / 100)
}

// Create a new container
func (e *Engine) Create(config *ContainerConfig, name string, pullImage bool, authConfig *dockerclient.AuthConfig) (*Container, error) {
	var (
		err    error
		id     string
		client = e.client
	)

	// Convert our internal ContainerConfig into something Docker will
	// understand.  Start by making a copy of the internal ContainerConfig as
	// we don't want to mess with the original.
	dockerConfig := config.ContainerConfig

	// nb of CPUs -> real CpuShares

	// FIXME remove "duplicate" lines and move this to cluster/config.go
	dockerConfig.CpuShares = int64(math.Ceil(float64(config.CpuShares*1024) / float64(e.Cpus)))
	dockerConfig.HostConfig.CpuShares = dockerConfig.CpuShares

	id, err = client.CreateContainer(&dockerConfig, name, nil)
	e.CheckConnectionErr(err)
	if err != nil {
		// If the error is other than not found, abort immediately.
		if err != dockerclient.ErrImageNotFound || !pullImage {
			return nil, err
		}
		// Otherwise, try to pull the image...
		if err = e.Pull(config.Image, authConfig); err != nil {
			return nil, err
		}
		// ...And try again.
		id, err = client.CreateContainer(&dockerConfig, name, nil)
		e.CheckConnectionErr(err)
		if err != nil {
			return nil, err
		}
	}

	// Register the container immediately while waiting for a state refresh.
	// Force a state refresh to pick up the newly created container.
	e.refreshContainer(id, true)
	e.RefreshVolumes()
	e.RefreshNetworks()

	e.Lock()
	container := e.containers[id]
	e.Unlock()

	if container == nil {
		err = errors.New("Container created but refresh didn't report it back")
	}
	return container, err
}

// RemoveContainer removes a container from the engine.
func (e *Engine) RemoveContainer(container *Container, force, volumes bool) error {
	err := e.client.RemoveContainer(container.Id, force, volumes)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	// Remove the container from the state. Eventually, the state refresh loop
	// will rewrite this.
	e.Lock()
	defer e.Unlock()
	delete(e.containers, container.Id)

	return nil
}

// CreateNetwork creates a network in the engine
func (e *Engine) CreateNetwork(request *types.NetworkCreate) (*types.NetworkCreateResponse, error) {
	response, err := e.apiClient.NetworkCreate(context.TODO(), *request)
	e.CheckConnectionErr(err)

	e.RefreshNetworks()

	return &response, err
}

// CreateVolume creates a volume in the engine
func (e *Engine) CreateVolume(request *types.VolumeCreateRequest) (*Volume, error) {
	volume, err := e.apiClient.VolumeCreate(context.TODO(), *request)

	e.RefreshVolumes()
	e.CheckConnectionErr(err)

	if err != nil {
		return nil, err
	}
	return &Volume{Volume: volume, Engine: e}, nil

}

// Pull an image on the engine
func (e *Engine) Pull(image string, authConfig *dockerclient.AuthConfig) error {
	if !strings.Contains(image, ":") {
		image = image + ":latest"
	}
	err := e.client.PullImage(image, authConfig)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	// force refresh images
	e.RefreshImages()

	return nil
}

// Load an image on the engine
func (e *Engine) Load(reader io.Reader) error {
	err := e.client.LoadImage(reader)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	// force fresh images
	e.RefreshImages()

	return nil
}

// Import image
func (e *Engine) Import(source string, repository string, tag string, imageReader io.Reader) error {
	_, err := e.client.ImportImage(source, repository, tag, imageReader)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	// force fresh images
	e.RefreshImages()

	return nil
}

// RegisterEventHandler registers an event handler.
func (e *Engine) RegisterEventHandler(h EventHandler) error {
	if e.eventHandler != nil {
		return errors.New("event handler already set")
	}
	e.eventHandler = h
	return nil
}

// Containers returns all the containers in the engine.
func (e *Engine) Containers() Containers {
	e.RLock()
	containers := Containers{}
	for _, container := range e.containers {
		containers = append(containers, container)
	}
	e.RUnlock()
	return containers
}

// Images returns all the images in the engine
func (e *Engine) Images() Images {
	e.RLock()
	images := make(Images, len(e.images))
	copy(images, e.images)
	e.RUnlock()
	return images
}

// Networks returns all the networks in the engine
func (e *Engine) Networks() Networks {
	e.RLock()

	networks := Networks{}
	for _, network := range e.networks {
		networks = append(networks, network)
	}
	e.RUnlock()
	return networks
}

// Volumes returns all the volumes in the engine
func (e *Engine) Volumes() Volumes {
	e.RLock()

	volumes := Volumes{}
	for _, volume := range e.volumes {
		volumes = append(volumes, volume)
	}
	e.RUnlock()
	return volumes
}

// Image returns the image with IDOrName in the engine
func (e *Engine) Image(IDOrName string) *Image {
	e.RLock()
	defer e.RUnlock()

	for _, image := range e.images {
		if image.Match(IDOrName, true) {
			return image
		}
	}
	return nil
}

func (e *Engine) String() string {
	return fmt.Sprintf("engine %s addr %s", e.ID, e.Addr)
}

func (e *Engine) handler(ev *dockerclient.Event, _ chan error, args ...interface{}) {
	// Something changed - refresh our internal state.

	switch ev.Type {
	case "network":
		e.RefreshNetworks()
	case "volume":
		e.RefreshVolumes()
	case "image":
		e.RefreshImages()
	case "container":
		switch ev.Action {
		case "die", "kill", "oom", "pause", "start", "restart", "stop", "unpause", "rename":
			e.refreshContainer(ev.ID, true)
		default:
			e.refreshContainer(ev.ID, false)
		}
	case "":
		// docker < 1.10
		switch ev.Status {
		case "pull", "untag", "delete", "commit":
			// These events refer to images so there's no need to update
			// containers.
			e.RefreshImages()
		case "die", "kill", "oom", "pause", "start", "stop", "unpause", "rename":
			// If the container state changes, we have to do an inspect in
			// order to update container.Info and get the new NetworkSettings.
			e.refreshContainer(ev.ID, true)
			e.RefreshVolumes()
			e.RefreshNetworks()
		default:
			// Otherwise, do a "soft" refresh of the container.
			e.refreshContainer(ev.ID, false)
			e.RefreshVolumes()
			e.RefreshNetworks()
		}

	}

	// If there is no event handler registered, abort right now.
	if e.eventHandler == nil {
		return
	}

	event := &Event{
		Engine: e,
		Event:  *ev,
	}

	e.eventHandler.Handle(event)
}

// AddContainer injects a container into the internal state.
func (e *Engine) AddContainer(container *Container) error {
	e.Lock()
	defer e.Unlock()

	if _, ok := e.containers[container.Id]; ok {
		return errors.New("container already exists")
	}
	e.containers[container.Id] = container
	return nil
}

// addImage injects an image into the internal state.
func (e *Engine) addImage(image *Image) {
	e.Lock()
	defer e.Unlock()

	e.images = append(e.images, image)
}

// removeContainer removes a container from the internal state.
func (e *Engine) removeContainer(container *Container) error {
	e.Lock()
	defer e.Unlock()

	if _, ok := e.containers[container.Id]; !ok {
		return errors.New("container not found")
	}
	delete(e.containers, container.Id)
	return nil
}

// cleanupContainers wipes the internal container state.
func (e *Engine) cleanupContainers() {
	e.Lock()
	e.containers = make(map[string]*Container)
	e.Unlock()
}

// StartContainer starts a container
func (e *Engine) StartContainer(id string, hostConfig *dockerclient.HostConfig) error {
	err := e.client.StartContainer(id, hostConfig)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	// refresh container
	_, err = e.refreshContainer(id, true)
	return err
}

// RenameContainer renames a container
func (e *Engine) RenameContainer(container *Container, newName string) error {
	// send rename request
	err := e.client.RenameContainer(container.Id, newName)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	// refresh container
	_, err = e.refreshContainer(container.Id, true)
	return err
}

// BuildImage builds an image
func (e *Engine) BuildImage(buildImage *types.ImageBuildOptions) (io.ReadCloser, error) {
	resp, err := e.apiClient.ImageBuild(context.TODO(), *buildImage)
	e.CheckConnectionErr(err)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// TagImage tags an image
func (e *Engine) TagImage(IDOrName string, repo string, tag string, force bool) error {
	// send tag request to docker engine
	err := e.client.TagImage(IDOrName, repo, tag, force)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	// refresh image
	return e.RefreshImages()
}
