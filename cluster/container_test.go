package cluster

import (
	"testing"

	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestContainersGet(t *testing.T) {
	containers := Containers([]*Container{{
		Container: dockerclient.Container{
			Id:    "container1-id",
			Names: []string{"/container1-name1", "/container1-name2"},
		},
		Engine: &Engine{ID: "test-engine"},
		Config: BuildContainerConfig(dockerclient.ContainerConfig{
			Labels: map[string]string{
				"com.docker.swarm.id": "swarm1-id",
			},
		}),
	}, {
		Container: dockerclient.Container{
			Id:    "container2-id",
			Names: []string{"/con"},
		},
		Engine: &Engine{ID: "test-engine"},
		Config: BuildContainerConfig(dockerclient.ContainerConfig{
			Labels: map[string]string{
				"com.docker.swarm.id": "swarm2-id",
			},
		}),
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
	assert.Equal(t, cc.Id, "container2-id")
}
