package cluster

import (
	"sync"

	log "github.com/Sirupsen/logrus"
)

type Watchdog struct {
	l       sync.Mutex
	cluster Cluster
}

// Handle cluster callbacks
func (w *Watchdog) Handle(e *Event) error {
	// Skip non-swarm events.
	if e.From != "swarm" {
		return nil
	}

	switch e.Status {
	case "engine_disconnect":
		go w.rescheduleContainers(e.Engine)
	}

	return nil
}

func (w *Watchdog) rescheduleContainers(e *Engine) {
	w.l.Lock()
	defer w.l.Unlock()

	log.Infof("Node %s failed - rescheduling containers", e.ID)
	for _, c := range e.Containers() {
		// Skip containers which don't have an "always" reschedule policy.
		if c.Config.ReschedulePolicy() != "always" {
			log.Debugf("Skipping rescheduling of %s based on rescheduling policy", c.Id)
			continue
		}

		// Remove the container from the dead engine. If we don't, then both
		// the old and new one will show up in docker ps.
		// We have to do this before calling `CreateContainer`, otherwise it
		// will abort because the name is already taken.
		c.Engine.removeContainer(c)

		newContainer, err := w.cluster.CreateContainer(c.Config, c.Info.Name)

		if err != nil {
			log.Errorf("Failed to reschedule container %s (Swarm ID: %s): %v", c.Id, c.Config.SwarmID(), err)
			continue
		}

		log.Infof("Rescheduled container %s from %s to %s as %s (Swarm ID: %s)", c.Id, c.Engine.ID, newContainer.Engine.ID, newContainer.Id, c.Config.SwarmID())

		if c.Info.State.Running {
			if err := newContainer.Start(); err != nil {
				log.Errorf("Failed to start rescheduled container %s", newContainer.Id)
			}
		}
	}
}

func NewWatchdog(cluster Cluster) *Watchdog {
	log.Debugf("Watchdog enabled")
	w := &Watchdog{
		cluster: cluster,
	}
	cluster.RegisterEventHandler(w)
	return w
}
