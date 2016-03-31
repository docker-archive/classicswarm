package mesos

import (
	"testing"
	"time"

	"github.com/docker/engine-api/types"
	containertypes "github.com/docker/engine-api/types/container"
	networktypes "github.com/docker/engine-api/types/network"
	"github.com/docker/swarm/cluster"
	"github.com/stretchr/testify/assert"
)

func createAgent(t *testing.T, ID string, containers ...*cluster.Container) *agent {
	engOpts := &cluster.EngineOpts{
		RefreshMinInterval: time.Duration(30) * time.Second,
		RefreshMaxInterval: time.Duration(60) * time.Second,
		FailureRetry:       3,
	}
	engine := cluster.NewEngine(ID, 0, engOpts)
	engine.Name = ID
	engine.ID = ID

	for _, container := range containers {
		container.Engine = engine
		engine.AddContainer(container)
	}

	return newAgent("agent-"+ID, engine)
}

func TestContainerLookup(t *testing.T) {
	c := &Cluster{
		agents: make(map[string]*agent),
	}

	container1 := &cluster.Container{
		Container: types.Container{
			ID:    "container1-id",
			Names: []string{"/container1-name1", "/container1-name2"},
		},
		Config: cluster.BuildContainerConfig(containertypes.Config{
			Labels: map[string]string{
				"com.docker.swarm.mesos.task": "task1-id",
				"com.docker.swarm.mesos.name": "container1-name",
			},
		}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}),
	}

	container2 := &cluster.Container{
		Container: types.Container{
			ID:    "container2-id",
			Names: []string{"/con"},
		},
		Config: cluster.BuildContainerConfig(containertypes.Config{
			Labels: map[string]string{
				"com.docker.swarm.mesos.task": "task2-id",
				"com.docker.swarm.mesos.name": "con",
			},
		}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}),
	}

	container3 := &cluster.Container{
		Container: types.Container{
			ID:    "container3-id",
			Names: []string{"/container3-name"},
		},
		Config: cluster.BuildContainerConfig(containertypes.Config{}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}),
	}

	s := createAgent(t, "test-engine", container1, container2, container3)
	c.agents[s.id] = s

	// Hide container without `com.docker.swarm.mesos.task`
	assert.Equal(t, len(c.Containers()), 2)

	// Invalid lookup
	container, err := c.Container("invalid-id")
	assert.Nil(t, container)
	assert.NotNil(t, err)

	container, err = c.Container("")
	assert.Nil(t, container)
	assert.NotNil(t, err)

	// Container ID lookup.
	container, err = c.Container("container1-id")
	assert.NotNil(t, container)
	assert.Nil(t, err)

	// Container ID prefix lookup.
	container, err = c.Container("container1-")
	assert.NotNil(t, container)
	assert.Nil(t, err)

	container, err = c.Container("container")
	assert.Nil(t, container)
	assert.NotNil(t, err)

	// Container name lookup.
	container, err = c.Container("container1-name1")
	assert.NotNil(t, container)
	assert.Nil(t, err)

	container, err = c.Container("container1-name2")
	assert.NotNil(t, container)
	assert.Nil(t, err)

	// Container engine/name matching.
	container, err = c.Container("test-engine/container1-name1")
	assert.NotNil(t, container)
	assert.Nil(t, err)

	container, err = c.Container("test-engine/container1-name2")
	assert.NotNil(t, container)
	assert.Nil(t, err)

	// Get name before ID prefix
	container, err = c.Container("con")
	assert.NotNil(t, container)
	assert.Equal(t, container.ID, "container2-id")

}
