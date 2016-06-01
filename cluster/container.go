package cluster

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/engine-api/types"
	"github.com/docker/go-units"
)

// CheckpointTicker is exported
type CheckpointTicker struct {
	Version      int
	Checkpointed map[int]bool
	Ticker       bool
}

// Container is exported
type Container struct {
	types.Container

	Config           *ContainerConfig
	Info             types.ContainerJSON
	Engine           *Engine
	CheckpointTicker CheckpointTicker
}

// StateString returns a single string to describe state
func StateString(state *types.ContainerState) string {
	startedAt, _ := time.Parse(time.RFC3339Nano, state.StartedAt)
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

	if state.Checkpointed {
		return "checkpointed"
	}

	if startedAt.IsZero() {
		return "created"
	}

	return "exited"
}

// FullStateString returns human-readable description of the state
func FullStateString(state *types.ContainerState) string {
	startedAt, _ := time.Parse(time.RFC3339Nano, state.StartedAt)
	finishedAt, _ := time.Parse(time.RFC3339Nano, state.FinishedAt)
	checkpointedAt, _ := time.Parse(time.RFC3339Nano, state.CheckpointedAt)
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

	if state.Checkpointed {
		return fmt.Sprintf("Checkpointed %s ago", units.HumanDuration(time.Now().UTC().Sub(checkpointedAt)))
	}

	if startedAt.IsZero() {
		return "Created"
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

// CheckpointContainerTicker set a checkpoint ticker
func (c *Container) CheckpointContainerTicker(checkpointTime time.Duration) {
	var ticker = time.NewTicker(checkpointTime)
	var stopCh = make(chan bool)
	c.CheckpointTicker = CheckpointTicker{
		Checkpointed: make(map[int]bool),
		Version:      0,
		Ticker:       true,
	}

	go func() {
		c.Engine.WaitContainer(c.ID)
		log.Infof("wait %s stop", c.ID)
		stopCh <- true
	}()
	go func() {
		for {
			select {
			case <-ticker.C:
				c.CheckpointTicker.Checkpointed[c.CheckpointTicker.Version] = false
				criuOpts := types.CriuConfig{
					ImagesDirectory: filepath.Join(c.Engine.DockerRootDir, "checkpoint", c.ID, strconv.Itoa(c.CheckpointTicker.Version), "criu.image"),
					WorkDirectory:   filepath.Join(c.Engine.DockerRootDir, "checkpoint", c.ID, strconv.Itoa(c.CheckpointTicker.Version), "criu.work"),
					LeaveRunning:    true,
				}

				err := c.Engine.CheckpointContainer(c.ID, criuOpts)
				if err != nil {
					log.Errorf("Error to checkpoint %s, %s", c.ID, err)
				} else {
					log.Infof("checkpoint container %s,  version %d", c.ID, c.CheckpointTicker.Version)
				}
				c.CheckpointTicker.Checkpointed[c.CheckpointTicker.Version] = true
				c.CheckpointTicker.Version++
			case <-stopCh:
				ticker.Stop()
				c.CheckpointTicker.Ticker = false
				log.Infof("%s stop checkpoint", c.ID)
				return
			}
		}
	}()
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
