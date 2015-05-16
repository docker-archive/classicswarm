package swarm

import (
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func createEngine(t *testing.T, ID string, containers ...*cluster.Container) *cluster.Engine {
	engine := cluster.NewEngine(ID, 0)
	engine.Name = ID
	engine.ID = ID

	for _, container := range containers {
		container.Engine = engine
		engine.AddContainer(container)
	}

	return engine
}

func TestContainerLookup(t *testing.T) {
	c := &Cluster{
		engines: make(map[string]*cluster.Engine),
	}
	container1 := &cluster.Container{
		Container: dockerclient.Container{
			Id:    "container-id1",
			Names: []string{"/container1-name1", "/container1-name2"},
		},
		Config: cluster.BuildContainerConfig(dockerclient.ContainerConfig{
			Labels: map[string]string{
				"com.docker.swarm.id": "swarm-id1",
			},
		}),
	}

	container2 := &cluster.Container{
		Container: dockerclient.Container{
			Id:    "container-id2",
			Names: []string{"/con"},
		},
		Config: cluster.BuildContainerConfig(dockerclient.ContainerConfig{
			Labels: map[string]string{
				"com.docker.swarm.id": "swarm-id2",
			},
		}),
	}

	n := createEngine(t, "test-engine", container1, container2)
	c.engines[n.ID] = n

	// Invalid lookup
	assert.Nil(t, c.Container("invalid-id"))
	assert.Nil(t, c.Container(""))
	// Container ID lookup.
	assert.NotNil(t, c.Container("container-id1"))
	// Container ID prefix lookup.
	assert.NotNil(t, c.Container("container-"))
	// Container name lookup.
	assert.NotNil(t, c.Container("container1-name1"))
	assert.NotNil(t, c.Container("container1-name2"))
	// Container engine/name matching.
	assert.NotNil(t, c.Container("test-engine/container1-name1"))
	assert.NotNil(t, c.Container("test-engine/container1-name2"))
	// Swarm ID lookup.
	assert.NotNil(t, c.Container("swarm-id1"))
	// Swarm ID prefix lookup.
	assert.NotNil(t, c.Container("swarm-"))
	// Match name before ID prefix
	cc := c.Container("con")
	assert.NotNil(t, cc)
	assert.Equal(t, cc.Id, "container-id2")
}
