package cluster

import (
	"testing"

	"github.com/docker/engine-api/types"
	containertypes "github.com/docker/engine-api/types/container"
	networktypes "github.com/docker/engine-api/types/network"
	"github.com/stretchr/testify/assert"
)

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
	}, {
		Container: types.Container{
			ID:    "container3-id",
			Names: []string{"/container-dup"},
		},
		Engine: &Engine{ID: "test-engine"},
		Config: BuildContainerConfig(containertypes.Config{
			Labels: map[string]string{
				"com.docker.swarm.id": "swarm3-id",
			},
		}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}),
	}, {
		Container: types.Container{
			ID:    "container4-id",
			Names: []string{"/container-dup"},
		},
		Engine: &Engine{ID: "test-engine"},
		Config: BuildContainerConfig(containertypes.Config{
			Labels: map[string]string{
				"com.docker.swarm.id": "swarm4-id",
			},
		}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}),
	}})

	// Invalid lookup
	container, err := containers.Get("invalid-id")
	assert.Nil(t, container)
	assert.NotNil(t, err)

	container, err = containers.Get("")
	assert.Nil(t, container)
	assert.NotNil(t, err)

	// Container ID lookup.
	container, err = containers.Get("container1-id")
	assert.NotNil(t, container)
	assert.Nil(t, err)

	// Container ID prefix lookup.
	container, err = containers.Get("container1-")
	assert.NotNil(t, container)
	assert.Nil(t, err)

	container, err = containers.Get("container")
	assert.Nil(t, container)
	assert.NotNil(t, err)

	// Container name lookup.
	container, err = containers.Get("container1-name1")
	assert.NotNil(t, container)
	assert.Nil(t, err)

	container, err = containers.Get("container1-name2")
	assert.NotNil(t, container)
	assert.Nil(t, err)

	// Container engine/name matching.
	container, err = containers.Get("test-engine/container1-name1")
	assert.NotNil(t, container)
	assert.Nil(t, err)

	container, err = containers.Get("test-engine/container1-name2")
	assert.NotNil(t, container)
	assert.Nil(t, err)

	// Swarm ID lookup.
	container, err = containers.Get("swarm1-id")
	assert.NotNil(t, container)
	assert.Nil(t, err)

	// Swarm ID prefix lookup.
	container, err = containers.Get("swarm1-")
	assert.NotNil(t, container)
	assert.Nil(t, err)

	container, err = containers.Get("swarm")
	assert.Nil(t, container)
	assert.NotNil(t, err)

	// Same container name
	container, err = containers.Get("container-dup")
	assert.Nil(t, container)
	assert.NotNil(t, err)

	container, err = containers.Get("test-engine/container-dup")
	assert.Nil(t, container)
	assert.NotNil(t, err)

	// Get name before ID prefix
	container, err = containers.Get("con")
	assert.NotNil(t, container)
	assert.Equal(t, container.ID, "container2-id")
}
