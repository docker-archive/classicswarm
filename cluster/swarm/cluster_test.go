package swarm

import (
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func createNode(t *testing.T, ID string, containers ...dockerclient.Container) *Node {
	node := NewNode(ID, 0)
	node.name = ID
	node.id = ID

	for _, container := range containers {
		node.AddContainer(&cluster.Container{Container: container, Node: node})
	}

	return node
}

func TestContainerLookup(t *testing.T) {
	c := &Cluster{
		nodes: make(map[string]*Node),
	}
	container := dockerclient.Container{
		Id:    "container-id",
		Names: []string{"/container-name1", "/container-name2"},
	}

	n := createNode(t, "test-node", container)
	c.nodes[n.ID()] = n

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
