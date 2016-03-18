package cluster

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/engine-api/types"
	"github.com/docker/go-units"
)

// Container is exported
type Container struct {
	types.Container

	Config *ContainerConfig
	Info   types.ContainerJSON
	Engine *Engine
}

// StateString returns a single string to describe state
func StateString(state *types.ContainerState) string {
	if state.Running {
		if state.Paused {
			return "paused"
		}
		if state.Restarting {
			return "restarting"
		}
		return "running"
	}

	if state.Dead {
		return "dead"
	}

	return "exited"
}

// FullStateString returns human-readable description of the state
func FullStateString(state *types.ContainerState) string {
	startedAt, _ := time.Parse(time.RFC3339Nano, state.StartedAt)
	finishedAt, _ := time.Parse(time.RFC3339Nano, state.FinishedAt)
	if state.Running {
		if state.Paused {
			return fmt.Sprintf("Up %s (Paused)", units.HumanDuration(time.Now().UTC().Sub(startedAt)))
		}
		if state.Restarting {
			return fmt.Sprintf("Restarting (%d) %s ago", state.ExitCode, units.HumanDuration(time.Now().UTC().Sub(finishedAt)))
		}
		return fmt.Sprintf("Up %s", units.HumanDuration(time.Now().UTC().Sub(startedAt)))
	}

	if state.Dead {
		return "Dead"
	}

	if finishedAt.IsZero() {
		return ""
	}

	return fmt.Sprintf("Exited (%d) %s ago", state.ExitCode, units.HumanDuration(time.Now().UTC().Sub(finishedAt)))
}

// Refresh container
func (c *Container) Refresh() (*Container, error) {
	return c.Engine.refreshContainer(c.ID, true)
}

// Containers represents a list of containers
type Containers []*Container

// Get returns a container using its ID or Name
func (containers Containers) Get(IDOrName string) *Container {
	// Abort immediately if the name is empty.
	if len(IDOrName) == 0 {
		return nil
	}

	// Match exact or short Container ID.
	for _, container := range containers {
		if container.ID == IDOrName || stringid.TruncateID(container.ID) == IDOrName {
			return container
		}
	}

	// Match exact Swarm ID.
	for _, container := range containers {
		if swarmID := container.Config.SwarmID(); swarmID == IDOrName || stringid.TruncateID(swarmID) == IDOrName {
			return container
		}
	}

	candidates := []*Container{}

	// Match name, /name or engine/name.
	for _, container := range containers {
		found := false
		for _, name := range container.Names {
			if name == IDOrName || name == "/"+IDOrName || container.Engine.ID+name == IDOrName || container.Engine.Name+name == IDOrName {
				found = true
			}
		}
		if found {
			candidates = append(candidates, container)
		}
	}

	if size := len(candidates); size == 1 {
		return candidates[0]
	} else if size > 1 {
		return nil
	}

	// Match Container ID prefix.
	for _, container := range containers {
		if strings.HasPrefix(container.ID, IDOrName) {
			candidates = append(candidates, container)
		}
	}

	// Match Swarm ID prefix.
	for _, container := range containers {
		if strings.HasPrefix(container.Config.SwarmID(), IDOrName) {
			candidates = append(candidates, container)
		}
	}

	if len(candidates) == 1 {
		return candidates[0]
	}

	return nil
}
