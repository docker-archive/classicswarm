package swarm

import (
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func createNode(t *testing.T, ID string, containers ...dockerclient.Container) *cluster.Engine {
	node := cluster.NewEngine(ID, 0)
	node.Name = ID
	node.ID = ID

	for _, container := range containers {
		node.AddContainer(&cluster.Container{Container: container, Engine: node})
	}

	return node
}

func TestContainerLookup(t *testing.T) {
	c := &Cluster{
		nodes: make(map[string]*cluster.Engine),
	}
	container := dockerclient.Container{
		Id:    "container-id",
		Names: []string{"/container-name1", "/container-name2"},
	}

	n := createNode(t, "test-node", container)
	c.nodes[n.ID] = n

	// Invalid lookup
	assert.Nil(t, c.Container("invalid-id"))
	assert.Nil(t, c.Container(""))
	// Container ID lookup.
	assert.NotNil(t, c.Container("container-id"))
	// Container ID prefix lookup.
	assert.NotNil(t, c.Container("container-"))
	// Container name lookup.
	assert.NotNil(t, c.Container("container-name1"))
	assert.NotNil(t, c.Container("container-name2"))
	// Container node/name matching.
	assert.NotNil(t, c.Container("test-node/container-name1"))
	assert.NotNil(t, c.Container("test-node/container-name2"))
}
