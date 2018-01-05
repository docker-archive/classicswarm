package cluster

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	engineapi "github.com/docker/docker/client"
	engineapinop "github.com/docker/swarm/api/nopclient"
	"github.com/docker/swarm/swarmclient"
)

const (
	// Timeout for requests sent out to the engine.
	requestTimeout = 30 * time.Second

	// Threshold of delta duration between swarm manager and engine's systime
	thresholdTime = 2 * time.Second
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

var (
	stateText = map[engineState]string{
		statePending:      "Pending",
		stateUnhealthy:    "Unhealthy",
		stateHealthy:      "Healthy",
		stateDisconnected: "Disconnected",
		//stateMaintenance: "Maintenance",
	}

	// testErrImageNotFound is only used for testing
	testErrImageNotFound = errors.New("TEST_ERR_IMAGE_NOT_FOUND_SWARM")
)

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
	Cpus    int64
	Memory  int64
	Labels  map[string]string
	Version string

	stopCh          chan struct{}
	refreshDelayer  *delayer
	containers      map[string]*Container
	images          []*Image
	networks        map[string]*Network
	volumes         map[string]*Volume
	httpClient      *http.Client
	url             *url.URL
	apiClient       swarmclient.SwarmAPIClient
	eventHandler    EventHandler
	state           engineState
	lastError       string
	updatedAt       time.Time
	failureCount    int
	overcommitRatio int64
	opts            *EngineOpts
	eventsMonitor   *EventsMonitor
	DeltaDuration   time.Duration // swarm's systime - engine's systime
}

// NewEngine is exported
func NewEngine(addr string, overcommitRatio float64, opts *EngineOpts) *Engine {
	e := &Engine{
		Addr:            addr,
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
	// TODO(nishanttotla): return the proper client after checking connection
	if _, ok := e.apiClient.(*engineapi.Client); ok {
		return e.httpClient, e.url.Scheme, nil
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

	addr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return err
	}
	e.IP = addr.IP.String()

	// create the HTTP Client and URL
	httpClient, url, err := NewHTTPClientTimeout("tcp://"+e.Addr, config, time.Duration(requestTimeout), nil)
	if err != nil {
		return err
	}
	e.httpClient = httpClient
	e.url = url

	// Use HTTP Client created above to create docker/api client
	apiClient, err := engineapi.NewClient("tcp://"+e.Addr, "", e.httpClient, nil)
	if err != nil {
		return err
	}

	return e.ConnectWithClient(apiClient)
}

// StartMonitorEvents monitors events from the engine
func (e *Engine) StartMonitorEvents() {
	log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).Debug("Start monitoring events")
	ec := make(chan error)

	go func() {
		if err := <-ec; err != nil {
			log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).WithError(err).Error("error monitoring events, will restart")
			// failing node reconnect should use back-off strategy to avoid frequent reconnect
			retryInterval := e.getFailureCount() + 1
			// maximum retry interval of 10 seconds
			if retryInterval > 10 {
				retryInterval = 10
			}
			<-time.After(time.Duration(retryInterval) * time.Second)
			e.StartMonitorEvents()
		}
		close(ec)
	}()

	// this call starts a goroutine that collects events from the the engine, and returns an error to
	// ec (the error channel) in case an error occurs. The corresponding Stop() function tears this
	// down. The eventsMonitor itself is initialized using the apiClient and handler (defined below)
	// The handler function processes events as received from the engine and decides what to do based
	// on each event. Moreover, it also calls the eventHandler's Handle() function.
	e.eventsMonitor.Start(ec)
}

// ConnectWithClient is exported
func (e *Engine) ConnectWithClient(apiClient swarmclient.SwarmAPIClient) error {
	e.apiClient = apiClient
	e.eventsMonitor = NewEventsMonitor(e.apiClient, e.handler)

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

	// Do not check error as older daemon doesn't support this call.
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
	e.eventsMonitor.Stop()

	// close idle connections
	if _, ok := e.apiClient.(*engineapi.Client); ok {
		closeIdleConnections(e.httpClient)
	}
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
	_, okAPIClient := e.apiClient.(*engineapinop.NopClient)
	return !okAPIClient
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
func (e *Engine) setState(state engineState, lockSet bool) {
	if !lockSet {
		e.Lock()
		defer e.Unlock()
	}
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
func (e *Engine) setErrMsg(errMsg string, lockSet bool) {
	if !lockSet {
		e.Lock()
		defer e.Unlock()
	}
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
	e.setErrMsg(fmt.Sprintf("ID duplicated. %s shared by this node %s and another node %s", e.ID, e.Addr, otherAddr), false)
}

// Status returns the health status of the Engine: Healthy or Unhealthy
func (e *Engine) Status() string {
	e.RLock()
	defer e.RUnlock()
	return stateText[e.state]
}

// incFailureCount increases engine's failure count, and sets engine as unhealthy if threshold is crossed
func (e *Engine) incFailureCount(lockSet bool) {
	if !lockSet {
		e.Lock()
		defer e.Unlock()
	}
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

func (e *Engine) resetFailureCount(lockSet bool) {
	if !lockSet {
		e.Lock()
		defer e.Unlock()
	}
	e.failureCount = 0
}

func (e *Engine) CheckConnectionErr(err error) {
	e.checkConnectionErr(err, false)
}

// CheckConnectionErr checks error from client response and adjusts engine healthy indicators
func (e *Engine) checkConnectionErr(err error, lockSet bool) {
	if err == nil {
		e.setErrMsg("", lockSet)
		// If current state is unhealthy, change it to healthy
		if e.state == stateUnhealthy {
			log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).Infof("Engine came back to life after %d retries. Hooray!", e.getFailureCount())
			e.emitEvent("engine_reconnect")
			e.setState(stateHealthy, lockSet)
		}
		e.resetFailureCount(lockSet)
		return
	}

	if IsConnectionError(err) {
		// each connection refused instance may increase failure count so
		// engine can fail fast. Short engine freeze or network failure may result
		// in engine marked as unhealthy. If this causes unnecessary failure, engine
		// can track last error time. Only increase failure count if last error is
		// not too recent, e.g., last error is at least 1 seconds ago.
		e.incFailureCount(lockSet)
		// update engine error message
		e.setErrMsg(err.Error(), lockSet)
		return
	}
	// other errors may be ambiguous.
}

// EngineToContainerNode constructs types.ContainerNode from engine
func (e *Engine) EngineToContainerNode() *types.ContainerNode {
	return &types.ContainerNode{
		ID:        e.ID,
		IPAddress: e.IP,
		Addr:      e.Addr,
		Name:      e.Name,
		Cpus:      int(e.Cpus),
		Memory:    int64(e.Memory),
		Labels:    e.Labels,
	}
}

// Gather engine specs (CPU, memory, constraints, ...).
func (e *Engine) updateSpecs() error {
	ctx := context.Background()

	info, err := e.apiClient.Info(ctx)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	if info.NCPU == 0 || info.MemTotal == 0 {
		return fmt.Errorf("cannot get resources for this engine, make sure %s is a Docker Engine, not a Swarm manager", e.Addr)
	}

	v, err := e.apiClient.ServerVersion(ctx)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	// update server version
	e.Version = v.Version
	// update client version. docker/api handles backward compatibility where needed
	e.apiClient.NegotiateAPIVersion(ctx)

	e.Lock()
	defer e.Unlock()
	// Swarm/docker identifies engine by ID. Updating ID but not updating cluster
	// index will put the cluster into inconsistent state. If this happens, the
	// engine should be put to pending state for re-validation.
	// We concatenate the engine ID and the address because sometimes engine
	// IDs can be duplicated on different nodes (e.g. if a user has
	// accidentally copied their /etc/docker/daemon.json file onto another
	// node).
	infoID := info.ID + "|" + e.Addr
	if e.ID == "" {
		e.ID = infoID
	} else if e.ID != infoID {
		e.state = statePending
		message := fmt.Sprintf("Engine (ID: %s, Addr: %s) shows up with another ID:%s. Please remove it from cluster, it can be added back.", e.ID, e.Addr, infoID)
		e.lastError = message
		return errors.New(message)
	}

	// delta is an estimation of time difference between manager and engine
	// with adjustment of delays (Engine response delay + network delay + manager process delay).
	var delta time.Duration
	if info.SystemTime != "" {
		engineTime, _ := time.Parse(time.RFC3339Nano, info.SystemTime)
		delta = time.Now().UTC().Sub(engineTime)
	} else {
		// if no SystemTime in info response, we treat delta as 0.
		delta = time.Duration(0)
	}

	// If the servers are sync up on time, this delta might be the source of error
	// we set a threshold that to ignore this case.
	absDelta := delta
	if delta.Seconds() < 0 {
		absDelta = time.Duration(-1*delta.Seconds()) * time.Second
	}

	if absDelta < thresholdTime {
		e.DeltaDuration = 0
	} else {
		log.Warnf("Engine (ID: %s, Addr: %s) has unsynchronized systime with swarm, please synchronize it.", e.ID, e.Addr)
		e.DeltaDuration = delta
	}

	e.Name = info.Name
	e.Cpus = int64(info.NCPU)
	e.Memory = info.MemTotal

	e.Labels = map[string]string{}
	if info.Driver != "" {
		e.Labels["storagedriver"] = info.Driver
	}
	if info.KernelVersion != "" {
		e.Labels["kernelversion"] = info.KernelVersion
	}
	if info.OperatingSystem != "" {
		e.Labels["operatingsystem"] = info.OperatingSystem
	}
	if info.OSType != "" {
		e.Labels["ostype"] = info.OSType
	}
	for _, label := range info.Labels {
		kv := strings.SplitN(label, "=", 2)
		if len(kv) != 2 {
			message := fmt.Sprintf("Engine (ID: %s, Addr: %s) contains an invalid label (%s) not formatted as \"key=value\".", e.ID, e.Addr, label)
			return errors.New(message)
		}

		// If an engine managed by Swarm contains a label with key "node",
		// such as node=node1
		// `docker run -e constraint:node==node1 -d nginx` will not work,
		// since "node" in constraint will match node.Name instead of label.
		// Log warn message in this case.
		if kv[0] == "node" {
			log.Warnf("Engine (ID: %s, Addr: %s) contains a label (%s) with key of \"node\" which cannot be used in Swarm.", e.ID, e.Addr, label)
			continue
		}

		if value, exist := e.Labels[kv[0]]; exist {
			log.Warnf("Node (ID: %s, Addr: %s) already contains a label (%s) with key (%s), and Engine's label (%s) cannot override it.", e.ID, e.Addr, value, kv[0], kv[1])
		} else {
			e.Labels[kv[0]] = kv[1]
		}
	}
	return nil
}

// RemoveImage deletes an image from the engine.
func (e *Engine) RemoveImage(name string, force bool) ([]types.ImageDeleteResponseItem, error) {
	rmOpts := types.ImageRemoveOptions{
		Force:         force,
		PruneChildren: true,
	}
	dels, err := e.apiClient.ImageRemove(context.Background(), name, rmOpts)
	e.CheckConnectionErr(err)

	// ImageRemove is not atomic. Engine may have deleted some layers and still failed.
	// Swarm should still refresh images before returning an error
	e.RefreshImages()
	return dels, err
}

// RemoveNetwork removes a network from the engine.
func (e *Engine) RemoveNetwork(network *Network) error {
	err := e.apiClient.NetworkRemove(context.Background(), network.ID)
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
	err := e.apiClient.VolumeRemove(context.Background(), name, false)
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
	images, err := e.apiClient.ImageList(context.Background(), imgLstOpts)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}
	e.Lock()
	e.images = nil
	for _, image := range images {
		e.images = append(e.images, &Image{ImageSummary: image, Engine: e})
	}
	e.Unlock()
	return nil
}

// refreshNetwork refreshes single network on the engine.
func (e *Engine) refreshNetwork(ID string) error {
	network, err := e.apiClient.NetworkInspect(context.Background(), ID, types.NetworkInspectOptions{})
	e.CheckConnectionErr(err)
	if err != nil {
		if strings.Contains(err.Error(), "No such network") {
			e.Lock()
			delete(e.networks, ID)
			e.Unlock()
			return nil
		}
		return err
	}

	e.Lock()
	e.networks[ID] = &Network{NetworkResource: network, Engine: e}
	e.Unlock()

	return nil
}

// RefreshNetworks refreshes the list of networks on the engine.
func (e *Engine) RefreshNetworks() error {
	netLsOpts := types.NetworkListOptions{Filters: filters.NewArgs()}
	networks, err := e.apiClient.NetworkList(context.Background(), netLsOpts)
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
	volumesLsRsp, err := e.apiClient.VolumeList(context.Background(), filters.NewArgs())
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

// refreshVolume refreshes single volume on the engine.
func (e *Engine) refreshVolume(IDOrName string) error {
	volume, err := e.apiClient.VolumeInspect(context.Background(), IDOrName)
	e.CheckConnectionErr(err)
	if err != nil {
		if strings.Contains(err.Error(), "No such volume") {
			e.Lock()
			delete(e.volumes, IDOrName)
			e.Unlock()
			return nil
		}
		return err
	}

	e.Lock()
	e.volumes[volume.Name] = &Volume{Volume: volume, Engine: e}
	e.Unlock()

	return nil
}

// RefreshContainers will refresh the list and status of containers running on the engine. If `full` is
// true, each container will be inspected.
// FIXME: unexport this method after mesos scheduler stops using it directly
func (e *Engine) RefreshContainers(full bool) error {
	e.Lock()
	defer e.Unlock()
	opts := types.ContainerListOptions{
		All:  true,
		Size: false,
	}
	ctx, cancel := context.WithTimeout(context.TODO(), requestTimeout)
	defer cancel()
	containers, err := e.apiClient.ContainerList(ctx, opts)
	e.checkConnectionErr(err, true)
	if err != nil {
		return err
	}

	merged := make(map[string]*Container)
	for _, c := range containers {
		mergedUpdate, err := e.updateContainer(c, merged, full, true)
		if err != nil {
			log.WithFields(log.Fields{"name": e.Name, "id": e.ID}).Errorf("Unable to update state of container %q: %v", c.ID, err)
		} else {
			merged = mergedUpdate
		}
	}

	e.containers = merged

	return nil
}

// Refresh the status of a container running on the engine. If `full` is true,
// the container will be inspected.
func (e *Engine) refreshContainer(ID string, full bool) (*Container, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("id", ID)
	opts := types.ContainerListOptions{
		All:     true,
		Size:    false,
		Filters: filterArgs,
	}
	containers, err := e.apiClient.ContainerList(context.Background(), opts)
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

	_, err = e.updateContainer(containers[0], e.containers, full, false)
	e.RLock()
	container := e.containers[containers[0].ID]
	e.RUnlock()
	return container, err
}

func (e *Engine) updateContainer(c types.Container, containers map[string]*Container, full bool, lockSet bool) (map[string]*Container, error) {
	var container *Container

	if !lockSet {
		e.RLock()
	}
	if current, exists := e.containers[c.ID]; exists {
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
	if !lockSet {
		e.RUnlock()
	}

	c.Created = time.Unix(c.Created, 0).Add(e.DeltaDuration).Unix()

	// Update ContainerInfo.
	if full {
		info, err := e.apiClient.ContainerInspect(context.Background(), c.ID)
		e.checkConnectionErr(err, lockSet)
		if err != nil {
			return nil, err
		}
		// Convert the ContainerConfig from inspect into our own
		// cluster.ContainerConfig.

		// info.HostConfig.CPUShares = info.HostConfig.CPUShares * int64(e.Cpus) / 1024.0
		networkingConfig := networktypes.NetworkingConfig{
			EndpointsConfig: info.NetworkSettings.Networks,
		}
		container.Config = BuildContainerConfig(*info.Config, *info.HostConfig, networkingConfig)
		// FIXME remove "duplicate" line and move this to cluster/config.go
		container.Config.HostConfig.CPUShares = container.Config.HostConfig.CPUShares * e.Cpus / 1024.0

		// consider the delta duration between swarm and docker engine
		startedAt, _ := time.Parse(time.RFC3339Nano, info.State.StartedAt)
		finishedAt, _ := time.Parse(time.RFC3339Nano, info.State.FinishedAt)

		info.State.StartedAt = startedAt.Add(e.DeltaDuration).Format(time.RFC3339Nano)
		info.State.FinishedAt = finishedAt.Add(e.DeltaDuration).Format(time.RFC3339Nano)

		// Save the entire inspect back into the container.
		container.Info = info
	}

	// Update its internal state.
	if !lockSet {
		e.Lock()
	}
	container.Container = c
	containers[container.ID] = container
	if !lockSet {
		e.Unlock()
	}

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

		err = e.RefreshContainers(false)
		if err == nil {
			// Do not check error as older daemon doesn't support this call
			e.RefreshVolumes()
			e.RefreshNetworks()
			e.RefreshImages()
			err := e.UpdateNetworkContainers("", false)
			if err != nil {
				log.WithFields(log.Fields{"id": e.ID, "name": e.Name}).Debugf("Engine refresh succeeded, but network containers update failed: %s", err.Error())
			} else {
				log.WithFields(log.Fields{"id": e.ID, "name": e.Name}).Debugf("Engine update succeeded")
			}
		} else {
			log.WithFields(log.Fields{"id": e.ID, "name": e.Name}).Debugf("Engine refresh failed")
		}
	}
}

// UpdateNetworkContainers updates the list of containers attached to each network.
// This is required because the RefreshNetworks uses NetworkList which has stopped
// returning this information for recent API versions. Note that the container cache
// is used to obtain this information, which means that if the container cache isn't
// up to date for whatever reason, then the network information might be out of date
// as well. This needs to be considered while designing networking related functionality.
// Ideally, this function should only be used in the refresh loop, and called AFTER
// RefreshNetworks() because RefreshNetworks() will not populate network information
// correctly if container refresh hasn't taken place.
// The containerID argument can be provided along with a full argument can be used to
// limit the update as it concerns only one container instead of doing it for all
// containers. The idea is that sometimes only things about one container may have
// changed, and we don't need to run the loop over all containers.
func (e *Engine) UpdateNetworkContainers(containerID string, full bool) error {
	containerMap := make(map[string]*Container)
	if containerID != "" {
		ctr, err := e.refreshContainer(containerID, full)
		if err != nil {
			return err
		}
		// in case the container doesn't exist on the engine
		if ctr == nil {
			log.Warnf("container %s doesn't exist on the engine %s, so terminate updating network for it", containerID, e.Name)
			return nil
		}
		containerMap[containerID] = ctr
	}

	e.Lock()
	defer e.Unlock()

	// if a specific container is not being targeted, then an update for all containers
	// will take place
	if containerID == "" {
		containerMap = e.containers
	}

	for _, c := range containerMap {
		if c.NetworkSettings == nil {
			continue
		}
		for _, n := range c.NetworkSettings.Networks {
			if n.NetworkID == "" {
				continue
			}
			engineNetwork, ok := e.networks[n.NetworkID]
			if !ok {
				// it shouldn't be the case that a network which a container is connected to wasn't
				// even listed. Return an error when that happens.
				return fmt.Errorf("container %s connected to network %s but the network wasn't listed in the refresh loop", c.ID, n.NetworkID)
			}

			// extract container name
			ctrName := ""
			if len(c.Names) != 0 {
				if s := strings.Split(c.Names[0], "/"); len(s) > 1 {
					ctrName = s[1]
				} else {
					ctrName = s[0]
				}
			}
			// extract ip addresses
			ipv4address := ""
			ipv6address := ""
			if n.IPAddress != "" {
				ipv4address = n.IPAddress + "/" + strconv.Itoa(n.IPPrefixLen)
			}
			if n.GlobalIPv6Address != "" {
				ipv6address = n.GlobalIPv6Address + "/" + strconv.Itoa(n.GlobalIPv6PrefixLen)
			}
			// update network information
			if engineNetwork.Containers == nil {
				engineNetwork.Containers = make(map[string]types.EndpointResource)
			}
			engineNetwork.Containers[c.ID] = types.EndpointResource{
				Name:        ctrName,
				EndpointID:  n.EndpointID,
				MacAddress:  n.MacAddress,
				IPv4Address: ipv4address,
				IPv6Address: ipv6address,
			}
		}
	}
	return nil
}

func (e *Engine) emitEvent(event string) {
	// If there is no event handler registered, abort right now.
	if e.eventHandler == nil {
		return
	}
	ev := &Event{
		Message: events.Message{
			Status: event,
			From:   "swarm",
			Type:   "swarm",
			Action: event,
			Actor: events.Actor{
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
		r += c.Config.HostConfig.Memory
	}
	e.RUnlock()
	return r
}

// UsedCpus returns the sum of CPUs reserved by containers.
func (e *Engine) UsedCpus() int64 {
	var r int64
	e.RLock()
	for _, c := range e.containers {
		r += c.Config.HostConfig.CPUShares
	}
	e.RUnlock()
	return r
}

// TotalMemory returns the total memory + overcommit
func (e *Engine) TotalMemory() int64 {
	return e.Memory + (e.Memory * e.overcommitRatio / 100)
}

// TotalCpus returns the total cpus + overcommit
func (e *Engine) TotalCpus() int64 {
	return e.Cpus + (e.Cpus * e.overcommitRatio / 100)
}

// CreateContainer creates a new container
func (e *Engine) CreateContainer(config *ContainerConfig, name string, pullImage bool, authConfig *types.AuthConfig) (*Container, error) {
	var (
		err        error
		createResp container.ContainerCreateCreatedBody
	)

	// Convert our internal ContainerConfig into something Docker will
	// understand.  Start by making a copy of the internal ContainerConfig as
	// we don't want to mess with the original.
	dockerConfig := *config

	// nb of CPUs -> real CpuShares

	// FIXME remove "duplicate" lines and move this to cluster/config.go
	dockerConfig.HostConfig.CPUShares = int64(math.Ceil(float64(config.HostConfig.CPUShares*1024) / float64(e.Cpus)))

	createResp, err = e.apiClient.ContainerCreate(context.Background(), &dockerConfig.Config, &dockerConfig.HostConfig, &dockerConfig.NetworkingConfig, name)
	e.CheckConnectionErr(err)
	if err != nil {
		// If the error is other than not found, abort immediately.
		if (err != testErrImageNotFound && !engineapi.IsErrImageNotFound(err)) || !pullImage {
			return nil, err
		}
		// Otherwise, try to pull the image...
		if err = e.Pull(config.Image, authConfig, nil); err != nil {
			return nil, err
		}
		// ...And try again.
		createResp, err = e.apiClient.ContainerCreate(context.Background(), &dockerConfig.Config, &dockerConfig.HostConfig, &dockerConfig.NetworkingConfig, name)
		e.CheckConnectionErr(err)
		if err != nil {
			return nil, err
		}
	}

	// Register the container immediately while waiting for a state refresh.
	// Force a state refresh to pick up the newly created container.
	e.refreshContainer(createResp.ID, true)

	e.Lock()
	container := e.containers[createResp.ID]
	e.Unlock()

	if container == nil {
		err = errors.New("Container created but refresh didn't report it back")
	}
	return container, err
}

// RemoveContainer removes a container from the engine.
func (e *Engine) RemoveContainer(container *Container, force, volumes bool) error {
	opts := types.ContainerRemoveOptions{
		Force:         force,
		RemoveVolumes: volumes,
	}
	err := e.apiClient.ContainerRemove(context.Background(), container.ID, opts)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	// Remove the container from the state. Eventually, the state refresh loop
	// will rewrite this.
	e.Lock()
	defer e.Unlock()
	delete(e.containers, container.ID)

	return nil
}

// CreateNetwork creates a network in the engine
func (e *Engine) CreateNetwork(name string, request *types.NetworkCreate) (*types.NetworkCreateResponse, error) {
	response, err := e.apiClient.NetworkCreate(context.Background(), name, *request)
	e.CheckConnectionErr(err)
	if err != nil {
		return nil, err
	}

	e.refreshNetwork(response.ID)

	return &response, err
}

// CreateVolume creates a volume in the engine
func (e *Engine) CreateVolume(request *volume.VolumesCreateBody) (*types.Volume, error) {
	volume, err := e.apiClient.VolumeCreate(context.Background(), *request)
	e.CheckConnectionErr(err)
	if err != nil {
		return nil, err
	}

	e.refreshVolume(volume.Name)

	return &volume, err
}

// encodeAuthToBase64 serializes the auth configuration as JSON base64 payload
func encodeAuthToBase64(authConfig *types.AuthConfig) (string, error) {
	if authConfig == nil {
		return "", nil
	}
	buf, err := json.Marshal(*authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

// Pull an image on the engine
func (e *Engine) Pull(image string, authConfig *types.AuthConfig, callback func(msg JSONMessage)) error {
	encodedAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		return err
	}
	pullOpts := types.ImagePullOptions{
		All:           false,
		RegistryAuth:  encodedAuth,
		PrivilegeFunc: nil,
	}
	// image is a ref here
	pullResponseBody, err := e.apiClient.ImagePull(context.Background(), image, pullOpts)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	defer pullResponseBody.Close()

	scanner := bufio.NewScanner(pullResponseBody)
	for scanner.Scan() {
		line := scanner.Text()
		var msg JSONMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Warnf("Malformed progress line during pull of %s: %s - %s", image, line, err)
			continue
		}

		if msg.Error != nil {
			return fmt.Errorf("Failed to load %s: %s", image, msg.Error.Message)
		}

		if callback != nil {
			callback(msg)
		}
	}

	// force refresh images
	e.RefreshImages()
	return nil
}

// Load an image on the engine
func (e *Engine) Load(reader io.Reader, callback func(msg JSONMessage)) error {
	loadResponse, err := e.apiClient.ImageLoad(context.Background(), reader, false)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	defer loadResponse.Body.Close()

	scanner := bufio.NewScanner(loadResponse.Body)
	for scanner.Scan() {
		line := scanner.Text()
		var msg JSONMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Warnf("Malformed progress line during image load: %s - %s", line, err)
			continue
		}

		if msg.Error != nil {
			return fmt.Errorf("Failed to load image: %s", msg.Error.Message)
		}

		if callback != nil {
			callback(msg)
		}
	}

	// force fresh images
	e.RefreshImages()

	return nil
}

// Import image
func (e *Engine) Import(source string, ref string, tag string, imageReader io.Reader, callback func(msg JSONMessage)) error {
	importSrc := types.ImageImportSource{
		Source:     imageReader,
		SourceName: source,
	}
	opts := types.ImageImportOptions{
		Tag: tag,
	}

	importResponseBody, err := e.apiClient.ImageImport(context.Background(), importSrc, ref, opts)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	defer importResponseBody.Close()

	scanner := bufio.NewScanner(importResponseBody)
	for scanner.Scan() {
		line := scanner.Text()
		var msg JSONMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Warnf("Malformed progress line during import of %s: %s - %s", ref, line, err)
			continue
		}

		if msg.Error != nil {
			return fmt.Errorf("Failed to import %s: %s", ref, msg.Error.Message)
		}

		if callback != nil {
			callback(msg)
		}
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

func (e *Engine) handler(msg events.Message) error {
	// Something changed - refresh our internal state.
	// TODO(nishanttotla): Possibly container events should trigger a container refresh

	switch msg.Type {
	case "network":
		e.refreshNetwork(msg.Actor.ID)
	case "volume":
		e.refreshVolume(msg.Actor.ID)
	case "image":
		e.RefreshImages()
	case "container":
		action := msg.Action
		// healthcheck events are like 'health_status: unhealthy'
		if strings.HasPrefix(action, "health_status") {
			action = "health_status"
		}
		switch action {
		case "commit":
			// commit a container will generate a new image
			e.RefreshImages()
		case "die", "kill", "oom", "pause", "start", "restart", "stop", "unpause", "rename", "update", "health_status":
			e.refreshContainer(msg.ID, true)
		case "top", "resize", "export", "exec_create", "exec_start", "exec_detach", "attach", "detach", "extract-to-dir", "copy", "archive-path":
			// no action needed
		default:
			e.refreshContainer(msg.ID, false)
		}
	case "daemon":
		// docker 1.12 started to support daemon events
		// https://github.com/docker/docker/pull/22590
		switch msg.Action {
		case "reload":
			e.updateSpecs()
		}
	case "":
		// docker < 1.10
		switch msg.Status {
		case "pull", "untag", "delete", "commit":
			// These events refer to images so there's no need to update
			// containers.
			e.RefreshImages()
		case "die", "kill", "oom", "pause", "start", "stop", "unpause", "rename":
			// If the container state changes, we have to do an inspect in
			// order to update container.Info and get the new NetworkSettings.
			e.refreshContainer(msg.ID, true)
			e.RefreshVolumes()
			e.RefreshNetworks()
		case "top", "resize", "export", "exec_create", "exec_start", "attach", "extract-to-dir", "copy", "archive-path":
			// no action needed
		default:
			// Otherwise, do a "soft" refresh of the container.
			e.refreshContainer(msg.ID, false)
			e.RefreshVolumes()
			e.RefreshNetworks()
		}

	}

	// If there is no event handler registered, abort right now.
	if e.eventHandler == nil {
		return nil
	}

	event := &Event{
		Engine:  e,
		Message: msg,
	}

	return e.eventHandler.Handle(event)
}

// AddContainer injects a container into the internal state.
func (e *Engine) AddContainer(container *Container) error {
	e.Lock()
	defer e.Unlock()

	if _, ok := e.containers[container.ID]; ok {
		return errors.New("container already exists")
	}
	e.containers[container.ID] = container
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

	if _, ok := e.containers[container.ID]; !ok {
		return errors.New("container not found")
	}
	delete(e.containers, container.ID)
	return nil
}

// cleanupContainers wipes the internal container state.
func (e *Engine) cleanupContainers() {
	e.Lock()
	e.containers = make(map[string]*Container)
	e.Unlock()
}

// StartContainer starts a container
func (e *Engine) StartContainer(container *Container) error {
	// TODO(nishanttotla): Should ContainerStartOptions be provided?
	err := e.apiClient.ContainerStart(context.Background(), container.ID, types.ContainerStartOptions{})
	e.CheckConnectionErr(err)

	if err != nil {
		return err
	}

	// refresh the container in the cache
	_, err = e.refreshContainer(container.ID, true)

	// If we could not inspect the container that was just started,
	// this indicates that it's been already removed by the daemon.
	// This is expected to occur in API versions 1.25 or higher if
	// the HostConfig.AutoRemove field is set to true. This could also occur
	// during race conditions where a third-party client removes the container
	// immediately after it's started.
	if container.Info.HostConfig.AutoRemove && engineapi.IsErrContainerNotFound(err) {
		delete(e.containers, container.ID)
		log.Debugf("container %s was not detected shortly after ContainerStart, indicating a daemon-side removal", container.ID)
		return nil
	}

	return err
}

// InspectContainer inspects a container
func (e *Engine) InspectContainer(id string) (*types.ContainerJSON, error) {
	container, err := e.apiClient.ContainerInspect(context.Background(), id)
	e.CheckConnectionErr(err)
	if err != nil {
		return nil, err
	}

	return &container, nil
}

// CreateContainerExec creates a container exec
func (e *Engine) CreateContainerExec(id string, config types.ExecConfig) (types.IDResponse, error) {
	execCreateResp, err := e.apiClient.ContainerExecCreate(context.Background(), id, config)
	e.CheckConnectionErr(err)
	if err != nil {
		return types.IDResponse{}, err
	}

	return execCreateResp, nil
}

// RenameContainer renames a container
func (e *Engine) RenameContainer(container *Container, newName string) error {
	// send rename request
	err := e.apiClient.ContainerRename(context.Background(), container.ID, newName)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	// refresh container
	_, err = e.refreshContainer(container.ID, true)
	return err
}

// BuildImage builds an image
func (e *Engine) BuildImage(buildContext io.Reader, buildImage *types.ImageBuildOptions, callback func(msg JSONMessage)) error {
	resp, err := e.apiClient.ImageBuild(context.Background(), buildContext, *buildImage)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		var msg JSONMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Warnf("Malformed progress line during image build: %s - %s", line, err)
			continue
		}

		if msg.Error != nil {
			return fmt.Errorf("Failed to build image: %s", msg.Error.Message)
		}

		if callback != nil {
			callback(msg)
		}
	}
	return nil
}

// TagImage tags an image
func (e *Engine) TagImage(IDOrName string, ref string, force bool) error {
	// send tag request to docker engine
	err := e.apiClient.ImageTag(context.Background(), IDOrName, ref)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}

	// refresh image
	return e.RefreshImages()
}

// NetworkDisconnect disconnects a container from a network
func (e *Engine) NetworkDisconnect(container *Container, network string, force bool) error {
	err := e.apiClient.NetworkDisconnect(context.Background(), network, container.ID, force)
	e.CheckConnectionErr(err)
	if err != nil {
		return err
	}
	err = e.RefreshNetworks()
	if err != nil {
		return err
	}
	return e.UpdateNetworkContainers(container.ID, true)
}

//IsConnectionError returns true when err is connection problem
func IsConnectionError(err error) bool {
	// dockerclient defines ErrConnectionRefused error. but if http client is from swarm, it's not using
	// dockerclient. We need string matching for these cases. Remove the first character to deal with
	// case sensitive issue.
	// docker/api returns ErrConnectionFailed error, so we check for that as long as dockerclient exists
	return engineapi.IsErrConnectionFailed(err) ||
		strings.Contains(err.Error(), "onnection refused") ||
		strings.Contains(err.Error(), "annot connect to the docker engine endpoint") ||
		strings.Contains(err.Error(), "annot connect to the Docker daemon") ||
		strings.Contains(err.Error(), "no route to host")
}

// RefreshEngine refreshes container records from an engine
func (e *Engine) RefreshEngine(hostname string) error {
	if hostname != e.Name {
		return fmt.Errorf("invalid engine name during refresh: %s vs %s", hostname, e.Name)
	}
	return e.RefreshContainers(true)
}
