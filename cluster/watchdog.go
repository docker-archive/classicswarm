package cluster

import (
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types/network"
	"golang.org/x/net/context"
)

// Watchdog listens to cluster events and handles container rescheduling
type Watchdog struct {
	sync.Mutex
	cluster Cluster
}

// Handle handles cluster callbacks
func (w *Watchdog) Handle(e *Event) error {
	// Skip non-swarm events.
	if e.From != "swarm" {
		return nil
	}

	switch e.Status {
	case "engine_connect", "engine_reconnect":
		go w.removeDuplicateContainers(e.Engine)
	case "engine_disconnect":
		go w.rescheduleContainers(e.Engine)
	}
	return nil
}

// removeDuplicateContainers removes duplicate containers when a node comes back
func (w *Watchdog) removeDuplicateContainers(e *Engine) {
	log.Debugf("removing duplicate containers from Node %s", e.ID)

	e.RefreshContainers(false)

	w.Lock()
	defer w.Unlock()

	for _, container := range e.Containers() {
		// skip non-swarm containers
		if container.Config.SwarmID() == "" {
			continue
		}

		for _, containerInCluster := range w.cluster.Containers() {
			if containerInCluster.Config.SwarmID() == container.Config.SwarmID() && containerInCluster.Engine.ID != container.Engine.ID {
				log.Debugf("container %s was rescheduled on node %s, removing it", container.ID, containerInCluster.Engine.Name)
				// container already exists in the cluster, destroy it
				if err := e.RemoveContainer(container, true, true); err != nil {
					log.Errorf("Failed to remove duplicate container %s on node %s: %v", container.ID, containerInCluster.Engine.Name, err)
				}
			}
		}
	}
}

// rescheduleContainers reschedules containers as soon as a node fails
func (w *Watchdog) rescheduleContainers(e *Engine) {
	w.Lock()
	defer w.Unlock()

	log.Debugf("Node %s failed - rescheduling containers", e.ID)

	for _, c := range e.Containers() {

		// Skip containers which don't have an "on-node-failure" reschedule policy.
		if !c.Config.HasReschedulePolicy("on-node-failure") {
			log.Debugf("Skipping rescheduling of %s based on rescheduling policies", c.ID)
			continue
		}

		// Remove the container from the dead engine. If we don't, then both
		// the old and new one will show up in docker ps.
		// We have to do this before calling `CreateContainer`, otherwise it
		// will abort because the name is already taken.
		c.Engine.removeContainer(c)

		// keep track of all global networks this container is connected to
		globalNetworks := make(map[string]*network.EndpointSettings)
		// if the existing container has global network endpoints,
		// they need to be removed with force option
		// "docker network disconnect -f network containername" only takes containername
		name := c.Info.Name
		if len(name) == 0 || len(name) == 1 && name[0] == '/' {
			log.Errorf("container %s has no name", c.ID)
			continue
		}
		// cut preceeding '/'
		if name[0] == '/' {
			name = name[1:]
		}

		if c.Info.NetworkSettings != nil && len(c.Info.NetworkSettings.Networks) > 0 {
			// find an engine to do disconnect work
			randomEngine, err := w.cluster.RANDOMENGINE()
			if err != nil {
				log.Errorf("Failed to find an engine to do network cleanup for container %s: %v", c.ID, err)
				// add the container back, so we can retry later
				c.Engine.AddContainer(c)
				continue
			}

			clusterNetworks := w.cluster.Networks().Uniq()
			for networkName, endpoint := range c.Info.NetworkSettings.Networks {
				net := clusterNetworks.Get(endpoint.NetworkID)
				if net != nil && (net.Scope == "global" || net.Scope == "swarm") {
					// record the network, they should be reconstructed on the new container
					globalNetworks[networkName] = endpoint
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					err = randomEngine.apiClient.NetworkDisconnect(ctx, networkName, name, true)
					if err != nil {
						// do not abort here as this endpoint might have been removed before
						log.Warnf("Failed to remove network endpoint from old container %s: %v", name, err)
					}
				}
			}
		}

		// Clear out the network configs that we're going to reattach
		// later.
		endpointsConfig := map[string]*network.EndpointSettings{}
		for k, v := range c.Config.NetworkingConfig.EndpointsConfig {
			net := w.cluster.Networks().Uniq().Get(v.NetworkID)
			if net != nil && (net.Scope == "global" || net.Scope == "swarm") {
				// These networks are already in globalNetworks
				// and thus will be reattached later.
				continue
			}
			endpointsConfig[k] = v
		}
		c.Config.NetworkingConfig.EndpointsConfig = endpointsConfig
		newContainer, err := w.cluster.CreateContainer(c.Config, c.Info.Name, nil)
		if err != nil {
			log.Errorf("Failed to reschedule container %s: %v", c.ID, err)
			// add the container back, so we can retry later
			c.Engine.AddContainer(c)
			continue
		}

		// Docker create command cannot create a container with multiple networks
		// see https://github.com/docker/docker/issues/17750
		// Add the global networks one by one
		for networkName, endpoint := range globalNetworks {
			hasSubnet := false
			network := w.cluster.Networks().Uniq().Get(networkName)
			if network != nil {
				for _, config := range network.IPAM.Config {
					if config.Subnet != "" {
						hasSubnet = true
						break
					}
				}
			}
			// If this network did not have a defined subnet, we
			// cannot connect to it with an explicit IP address.
			if !hasSubnet && endpoint.IPAMConfig != nil {
				endpoint.IPAMConfig.IPv4Address = ""
				endpoint.IPAMConfig.IPv6Address = ""
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err = newContainer.Engine.apiClient.NetworkConnect(ctx, networkName, name, endpoint)
			if err != nil {
				log.Warnf("Failed to connect network %s to container %s: %v", networkName, name, err)
			}
		}

		log.Infof("Rescheduled container %s from %s to %s as %s", c.ID, c.Engine.Name, newContainer.Engine.Name, newContainer.ID)
		if c.Info.State.Running {
			log.Infof("Container %s was running, starting container %s", c.ID, newContainer.ID)
			if err := w.cluster.StartContainer(newContainer, nil); err != nil {
				log.Errorf("Failed to start rescheduled container %s: %v", newContainer.ID, err)
			}
		}
	}
}

// NewWatchdog creates a new watchdog
func NewWatchdog(cluster Cluster) *Watchdog {
	log.Debugf("Watchdog enabled")
	w := &Watchdog{
		cluster: cluster,
	}
	cluster.RegisterEventHandler(w)
	return w
}
