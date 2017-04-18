package cluster

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/stretchr/testify/assert"
)

func TestStateString(t *testing.T) {
	states := map[string]*types.ContainerState{
		"paused": {
			Running: true,
			Paused:  true,
		},
		"restarting": {
			Running:    true,
			Restarting: true,
		},
		"running": {
			Running: true,
		},
		"dead": {
			Dead: true,
		},
		"created": {
			StartedAt: "",
		},
		"exited": {
			StartedAt: "2016-11-23T15:44:05.999999991Z",
		},
	}

	for k, s := range states {
		r := StateString(s)

		assert.Equal(t, k, r)
	}
}

func TestFullStateString(t *testing.T) {
	states := map[string]*types.ContainerState{
		"Up Less than a second (Paused)": {
			Running:   true,
			Paused:    true,
			StartedAt: time.Now().Format(time.RFC3339Nano),
		},
		"Restarting (10) Less than a second ago": {
			Running:    true,
			Restarting: true,
			ExitCode:   10,
			FinishedAt: time.Now().Format(time.RFC3339Nano),
		},
		"Up Less than a second": {
			Running:   true,
			StartedAt: time.Now().Format(time.RFC3339Nano),
		},
		"Dead": {
			Dead: true,
		},
		"Created": {
			StartedAt: "",
		},
		"": {
			StartedAt:  "2016-11-23T15:44:05.999999991Z",
			FinishedAt: "",
		},
		"Exited (10) Less than a second ago": {
			ExitCode:   10,
			StartedAt:  "2016-11-23T15:44:05.999999991Z",
			FinishedAt: time.Now().Format(time.RFC3339Nano),
		},
	}

	for k, s := range states {
		r := FullStateString(s)

		assert.Equal(t, k, r)
	}
}

func TestContainersGet(t *testing.T) {
	containers := Containers([]*Container{{
		Container: types.Container{
			ID:    "container1-id",
			Names: []string{"/container1-name1", "/container1-name2"},
		},
		Engine: &Engine{ID: "test-engine"},
		Config: BuildContainerConfig(containertypes.Config{
			Labels: map[string]string{
				"com.docker.swarm.id": "swarm1-id",
			},
		}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}),
	}, {
		Container: types.Container{
			ID:    "container2-id",
			Names: []string{"/con"},
		},
		Engine: &Engine{ID: "test-engine"},
		Config: BuildContainerConfig(containertypes.Config{
			Labels: map[string]string{
				"com.docker.swarm.id": "swarm2-id",
			},
		}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}),
	}})

	// Invalid lookup
	assert.Nil(t, containers.Get("invalid-id"))
	assert.Nil(t, containers.Get(""))
	// Container ID lookup.
	assert.NotNil(t, containers.Get("container1-id"))
	// Container ID prefix lookup.
	assert.NotNil(t, containers.Get("container1-"))
	assert.Nil(t, containers.Get("container"))
	// Container name lookup.
	assert.NotNil(t, containers.Get("container1-name1"))
	assert.NotNil(t, containers.Get("container1-name2"))
	// Container engine/name matching.
	assert.NotNil(t, containers.Get("test-engine/container1-name1"))
	assert.NotNil(t, containers.Get("test-engine/container1-name2"))
	// Swarm ID lookup.
	assert.NotNil(t, containers.Get("swarm1-id"))
	// Swarm ID prefix lookup.
	assert.NotNil(t, containers.Get("swarm1-"))
	assert.Nil(t, containers.Get("swarm"))
	// Get name before ID prefix
	cc := containers.Get("con")
	assert.NotNil(t, cc)
	assert.Equal(t, cc.ID, "container2-id")

}
