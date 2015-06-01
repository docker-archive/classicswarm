package mesos

import (
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func createSlave(t *testing.T, ID string, containers ...*cluster.Container) *slave {
	engine := cluster.NewEngine(ID, 0)
	engine.Name = ID
	engine.ID = ID

	for _, container := range containers {
		container.Engine = engine
		engine.AddContainer(container)
	}

	return newSlave("slave-"+ID, engine)
}

func TestContainerLookup(t *testing.T) {
	c := &Cluster{
		slaves: make(map[string]*slave),
	}
	container1 := &cluster.Container{
		Container: dockerclient.Container{
			Id:    "container1-id",
			Names: []string{"/container1-name1", "/container1-name2"},
		},
		Config: cluster.BuildContainerConfig(dockerclient.ContainerConfig{
			Labels: map[string]string{
				"com.docker.swarm.id": "swarm1-id",
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
				"com.docker.swarm.id": "swarm2-id",
			},
		}),
	}

	s := createSlave(t, "test-engine", container1, container2)
	c.slaves[s.id] = s

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
	// Swarm ID lookup.
	assert.NotNil(t, c.Container("swarm1-id"))
	// Swarm ID prefix lookup.
	assert.NotNil(t, c.Container("swarm1-"))
	assert.Nil(t, c.Container("swarm"))
	// Match name before ID prefix
	cc := c.Container("con")
	assert.NotNil(t, cc)
	assert.Equal(t, cc.Id, "container2-id")
}
