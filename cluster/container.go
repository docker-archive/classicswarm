package cluster

import (
	"fmt"
	"time"

	"github.com/docker/docker/pkg/units"
	"github.com/samalba/dockerclient"
)

// Container is exported
type Container struct {
	dockerclient.Container

	Config *ContainerConfig
	Info   dockerclient.ContainerInfo
	Engine *Engine
}

// Status returns a human-readable description of the state
// Stoken from docker/docker/daemon/state.go
func (c *Container) Status() string {
	s := c.Info.State
	if s.Running {
		if s.Paused {
			return fmt.Sprintf("Up %s (Paused)", units.HumanDuration(time.Now().UTC().Sub(s.StartedAt)))
		}
		if s.Restarting {
			return fmt.Sprintf("Restarting (%d) %s ago", s.ExitCode, units.HumanDuration(time.Now().UTC().Sub(s.FinishedAt)))
		}

		return fmt.Sprintf("Up %s", units.HumanDuration(time.Now().UTC().Sub(s.StartedAt)))
	}

	if s.Dead {
		return "Dead"
	}

	if s.FinishedAt.IsZero() {
		return ""
	}

	return fmt.Sprintf("Exited (%d) %s ago", s.ExitCode, units.HumanDuration(time.Now().UTC().Sub(s.FinishedAt)))
}

// StateString returns a single string to describe state
// Stoken from docker/docker/daemon/state.go
func (c *Container) StateString() string {
	s := c.Info.State
	if s.Running {
		if s.Paused {
			return "paused"
		}
		if s.Restarting {
			return "restarting"
		}
		return "running"
	}

	if s.Dead {
		return "dead"
	}

	return "exited"
}
