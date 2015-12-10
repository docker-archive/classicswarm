package mesos

import (
	"testing"
	"time"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
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
		Container: dockerclient.Container{
			Id:    "container1-id",
			Names: []string{"/container1-name1", "/container1-name2"},
		},
		Config: cluster.BuildContainerConfig(dockerclient.ContainerConfig{
			Labels: map[string]string{
				"com.docker.swarm.mesos.task": "task1-id",
				"com.docker.swarm.mesos.name": "container1-name1",
			},
		}),
	}

	container2 := &cluster.Container{
		Container: dockerclient.Container{
			Id:    "container2-id",
			Names: []string{"/con"},
		},
		Config: cluster.BuildContainerConfig(dockerclient.ContainerConfig{
			Labels: map[string]string{
				"com.docker.swarm.mesos.task": "task2-id",
				"com.docker.swarm.mesos.name": "con",
			},
		}),
	}

	container3 := &cluster.Container{
		Container: dockerclient.Container{
			Id:    "container3-id",
			Names: []string{"/container3-name"},
		},
		Config: cluster.BuildContainerConfig(dockerclient.ContainerConfig{}),
	}

	s := createAgent(t, "test-engine", container1, container2, container3)
	c.agents[s.id] = s

	// Hide container without `com.docker.swarm.mesos.task`
	assert.Equal(t, len(c.Containers()), 2)

	// Invalid lookup
	assert.Nil(t, c.Container("invalid-id"))
	assert.Nil(t, c.Container(""))
	// Container ID lookup.
	assert.NotNil(t, c.Container("container1-id"))
	// Container ID prefix lookup.
	assert.NotNil(t, c.Container("container1-"))
	assert.Nil(t, c.Container("container"))
	// Container name lookup.
	assert.NotNil(t, c.Container("container1-name1"))
	assert.NotNil(t, c.Container("container1-name2"))
	// Container engine/name matching.
	assert.NotNil(t, c.Container("test-engine/container1-name1"))
	assert.NotNil(t, c.Container("test-engine/container1-name2"))
	// Match name before ID prefix
	cc := c.Container("con")
	assert.NotNil(t, cc)
	assert.Equal(t, cc.Id, "container2-id")
}
