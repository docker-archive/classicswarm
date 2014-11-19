package strategy

import (
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func createNode(ID string, memory int64, cpus int) *cluster.Node {
	node := cluster.NewNode(ID, "")
	node.Memory = memory * 1024 * 1024 * 1024
	node.Cpus = cpus
	return node
}

func createConfig(memory int, cpus int) *dockerclient.ContainerConfig {
	return &dockerclient.ContainerConfig{Memory: memory * 1024 * 1024 * 1024, CpuShares: cpus}
}

func createContainer(ID string, config *dockerclient.ContainerConfig) *cluster.Container {
	return &cluster.Container{Container: dockerclient.Container{Id: ID}, Info: dockerclient.ContainerInfo{Config: config}}
}

func TestPlaceContainer(t *testing.T) {
	s := &BinPackingPlacementStrategy{}

	nodes := []*cluster.Node{
		createNode("node-1", 2, 4),
		createNode("node-2", 2, 4),
		createNode("node-3", 2, 4),
	}

	// try to place a 10G container
	config := createConfig(10, 1)
	_, err := s.PlaceContainer(config, nodes)

	// check that it refuses because the cluster is full
	assert.Error(t, err)

	// add one container 1G
	config = createConfig(1, 1)
	node1, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	node1.AddContainer(createContainer("c1", config))

	// add another container 1G
	config = createConfig(1, 1)
	node2, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	node2.AddContainer(createContainer("c2", config))

	// check that both containers ended on the same node
	assert.Equal(t, node1.ID, node2.ID, "")
	assert.Equal(t, len(node1.Containers()), len(node2.Containers()), "")

	// add another container 2G
	config = createConfig(2, 1)
	node3, err := s.PlaceContainer(createConfig(2, 1), nodes)
	assert.NoError(t, err)
	node3.AddContainer(createContainer("c3", config))

	// check that it ends up on another node
	assert.NotEqual(t, node1.ID, node3.ID, "")

	// add another container 1G
	config = createConfig(1, 1)
	node4, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	node4.AddContainer(createContainer("c4", config))

	// check that it ends up on another node
	assert.NotEqual(t, node1.ID, node4.ID, "")
	assert.NotEqual(t, node3.ID, node4.ID, "")

	// add another container 1G
	config = createConfig(1, 1)
	node5, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	node5.AddContainer(createContainer("c5", config))

	// check that it ends up on the same node
	assert.Equal(t, node4.ID, node5.ID, "")

	// try to add another container
	config = createConfig(1, 1)
	_, err = s.PlaceContainer(config, nodes)

	// check that it refuses because the cluster is full
	assert.Error(t, err)

	// remove container in the middle
	node3.CleanupContainers()

	// add another container
	config = createConfig(1, 1)
	node6, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	node6.AddContainer(createContainer("c6", config))

	// check it ends up on `node3`
	assert.Equal(t, node3.ID, node6.ID, "")
	assert.Equal(t, len(node3.Containers()), len(node6.Containers()), "")
}
