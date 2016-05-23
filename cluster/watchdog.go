package cluster

import (
	"sync"

	log "github.com/Sirupsen/logrus"
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
	case "engine_reconnect":
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

		// start to reschedule container
		w.rescheduleContainer(c)
	}
}

// rescheduleContainer Removes the container from the dead engine.
// If we don't, then both the old and new one will show up in docker ps.
// We have to do this before calling `CreateContainer`, otherwise it
// will abort because the name is already taken.
func (w *Watchdog) rescheduleContainer(c *Container) {
	c.Engine.removeContainer(c)

	newContainer, err := w.cluster.CreateContainer(c.Config, c.Info.Name, nil)
	if err != nil {
		log.Errorf("Failed to reschedule container %s: %v", c.ID, err)
		// add the container back, so we can retry later
		c.Engine.AddContainer(c)
		return
	}

	log.Infof("Rescheduled container %s from %s to %s as %s", c.ID, c.Engine.Name, newContainer.Engine.Name, newContainer.ID)
	w.setDesiredState(c, newContainer)
}

// setDesiredState sets desired state for the rescheduled container.
func (w *Watchdog) setDesiredState(c, newCon *Container) {
	// Original state of rescheduled container is running.
	// State Running can be divided into 3 types: Paused, Restarting, Running.
	state := StateString(c.Info.State)
	if c.Info.State.Running {
		log.Debugf("Container %s is running, start container %s", c.ID, newCon.ID)
		// no matter running, restarting or paused, start it first.
		if err := w.cluster.StartContainer(newCon, nil); err != nil {
			log.Errorf("Failed to start rescheduled container %s: %v", newCon.ID, err)
			return
		}

		// If container's state is Paused, we need to pause it.
		if state == "paused" {
			log.Debugf("Container %s is paused, pause container %s", c.ID, newCon.ID)
			if err := w.cluster.PauseContainer(newCon); err != nil {
				log.Errorf("Failed to pause rescheduled container %s: %v", newCon.ID, err)
				return
			}
		}
		return
	}

	// Original state of rescheduled container is dead or created.
	// Do nothing.
	if state == "dead" || state == "created" {
		return
	}

	// Original state of rescheduled container is exited.
	// Start and pause it.
	if state == "exited" {
		log.Debugf("Container %s is exited, start and pause container %s", c.ID, newCon.ID)
		if err := w.cluster.StartContainer(newCon, nil); err != nil {
			log.Errorf("Failed to start rescheduled container %s: %v", newCon.ID, err)
			return
		}

		if err := w.cluster.PauseContainer(newCon); err != nil {
			log.Errorf("Failed to pause rescheduled container %s: %v", newCon.ID, err)
			return
		}
	}

	return
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
