package cluster

import (
	"path/filepath"
	"strconv"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
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
		if !c.Config.HasReschedulePolicy("on-node-failure") && !c.Config.HasReschedulePolicy("restore") {
			log.Debugf("Skipping rescheduling of %s based on rescheduling policies", c.ID)
			continue
		}

		// Remove the container from the dead engine. If we don't, then both
		// the old and new one will show up in docker ps.
		// We have to do this before calling `CreateContainer`, otherwise it
		// will abort because the name is already taken.
		c.Engine.removeContainer(c)

		newContainer, err := w.cluster.CreateContainer(c.Config, c.Info.Name, nil)

		if err != nil {
			log.Errorf("Failed to reschedule container %s: %v", c.ID, err)
			// add the container back, so we can retry later
			c.Engine.AddContainer(c)
		} else {
			log.Infof("Rescheduled container %s from %s to %s as %s", c.ID, c.Engine.Name, newContainer.Engine.Name, newContainer.ID)
			if c.Info.State.Running {
				log.Infof("Container %s was running, starting container %s", c.ID, newContainer.ID)
				if c.Config.HasReschedulePolicy("on-node-failure") {
					if err := w.cluster.StartContainer(newContainer, nil); err != nil {
						log.Errorf("Failed to start rescheduled container %s: %v", newContainer.ID, err)
					}
				} else if c.Config.HasReschedulePolicy("restore") {
					w.restoreContainer(c, newContainer)
					if checkpointTime, err := c.Config.HasCheckpointTimePolicy(); err != nil {
						log.Errorf("Fails to set container %s checkpoint time, %s", c.ID, err)
					} else if checkpointTime > 0 {
						if c.CheckpointTicker.Ticker == false {
							c.CheckpointContainerTicker(checkpointTime)
						}
					}
				}
			}
		}
	}
}

func (w *Watchdog) restoreContainer(c *Container, newContainer *Container) {
	var err error
	for version := c.CheckpointTicker.Version; version >= 0 && version >= c.CheckpointTicker.Version-2; version-- {
		if c.CheckpointTicker.Checkpointed[version] != true {
			continue
		}
		criuOpts := types.CriuConfig{
			ImagesDirectory: filepath.Join(newContainer.Engine.DockerRootDir, "checkpoint", c.ID, strconv.Itoa(version), "criu.image"),
			WorkDirectory:   filepath.Join(newContainer.Engine.DockerRootDir, "checkpoint", c.ID, strconv.Itoa(version), "criu.work"),
		}
		if err = w.cluster.RestoreContainer(newContainer, criuOpts, true); err != nil {
			log.Errorf("Failed to restore rescheduled container %s version %d: %v", newContainer.ID, version, err)
		} else {
			log.Infof("restore %s to %s on version %d", c.ID, newContainer.ID, version)
			break
		}
	}
	//if restore fail 3 times, try to start a new container
	if err != nil {
		if err := w.cluster.StartContainer(newContainer, nil); err != nil {
			log.Errorf("Failed to start rescheduled container %s: %v", newContainer.ID, err)
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
