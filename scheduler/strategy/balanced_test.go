package strategy

import (
	"fmt"
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/stretchr/testify/assert"
)

func TestBalancedPlaceContainerMemory(t *testing.T) {
	s := &BalancedPlacementStrategy{}

	nodes := []*cluster.Node{}
	for i := 0; i < 2; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 2, 1))
	}

	// add 1 container 1G
	config := createConfig(1, 0)
	c1node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, c1node.AddContainer(createContainer("c1", config)))

	assert.Equal(t, c1node.ReservedMemory(), 1024*1024*1024)

	// add another container 1G
	config = createConfig(1, 1)
	c2node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, c2node.AddContainer(createContainer("c2", config)))

	assert.Equal(t, c2node.ReservedMemory(), 1024*1024*1024)

	// check that both containers ended on different nodes
	assert.NotEqual(t, c1node.ID, c2node.ID, "")
	assert.Equal(t, len(c1node.Containers()), len(c2node.Containers()), "")
}

func TestBalancedPlaceContainerCPU(t *testing.T) {
	s := &BalancedPlacementStrategy{}

	nodes := []*cluster.Node{}
	for i := 0; i < 2; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 1, 2))
	}

	// add 1 container 1CPU
	config := createConfig(0, 1)
	c1node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, c1node.AddContainer(createContainer("c1", config)))

	assert.Equal(t, c1node.ReservedCpus(), 1)

	// add another container 1CPU
	config = createConfig(0, 1)
	c2node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, c2node.AddContainer(createContainer("c2", config)))
	assert.Equal(t, c2node.ReservedCpus(), 1)

	// check that both containers ended on different nodes
	assert.NotEqual(t, c1node.ID, c2node.ID, "")
	assert.Equal(t, len(c1node.Containers()), len(c2node.Containers()), "")
}

func TestBalancedPlaceContainerHuge(t *testing.T) {
	s := &BalancedPlacementStrategy{}

	nodes := []*cluster.Node{}
	for i := 0; i < 100; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 1, 1))
	}

	// add 100 container 1CPU
	for i := 0; i < 100; i++ {
		node, err := s.PlaceContainer(createConfig(0, 1), nodes)
		assert.NoError(t, err)
		assert.NoError(t, node.AddContainer(createContainer(fmt.Sprintf("c%d", i), createConfig(0, 1))))
	}

	// add 100 container 1G
	for i := 100; i < 200; i++ {
		node, err := s.PlaceContainer(createConfig(1, 0), nodes)
		assert.NoError(t, err)
		assert.NoError(t, node.AddContainer(createContainer(fmt.Sprintf("c%d", i), createConfig(1, 0))))
	}
}

func TestBalancedPlaceContainerOvercommit(t *testing.T) {
	s, err := New("balanced")
	assert.NoError(t, err)

	nodes := []*cluster.Node{createNode("node-1", 0, 1)}
	nodes[0].Memory = 100

	config := createConfig(0, 0)

	// Below limit should still work.
	config.Memory = 90
	node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.Equal(t, node, nodes[0])

	// At memory limit should still work.
	config.Memory = 100
	node, err = s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.Equal(t, node, nodes[0])

	// Up to 105% it should still work.
	config.Memory = 105
	node, err = s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.Equal(t, node, nodes[0])

	// Above it should return an error.
	config.Memory = 106
	node, err = s.PlaceContainer(config, nodes)
	assert.Error(t, err)
}

// The demo
func TestBalancedPlaceContainerDemo(t *testing.T) {
	s := &BalancedPlacementStrategy{}

	nodes := []*cluster.Node{}
	for i := 0; i < 3; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 4, 4))
	}

	// try to place a 10G container
	config := createConfig(10, 0)
	_, err := s.PlaceContainer(config, nodes)

	// check that it refuses because no node has enough memory
	assert.Error(t, err)

	// add one container 1G
	config = createConfig(1, 0)
	c1node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, c1node.AddContainer(createContainer("c1", config)))

	// nodes: 1G 0CPU, 0G 0CPU, 0G 0CPU

	// add another container 1G
	config = createConfig(1, 0)
	c2node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, c2node.AddContainer(createContainer("c2", config)))

	// check that both containers ended on different nodes
	assert.NotEqual(t, c1node.ID, c2node.ID, "")
	assert.Equal(t, len(c1node.Containers()), len(c2node.Containers()), "")

	// nodes: 1G 0CPU, 1G 0CPU, 0G 0CPU

	// add another container 2G
	config = createConfig(2, 0)
	c3node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, c3node.AddContainer(createContainer("c3", config)))

	// check that it ends up on another node
	assert.NotEqual(t, c3node.ID, c1node.ID, "")
	assert.NotEqual(t, c3node.ID, c2node.ID, "")

	// nodes: 1G 0CPU, 1G 0CPU, 2G 0CPU

	// add another container 1G
	config = createConfig(1, 0)
	c4node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, c4node.AddContainer(createContainer("c4", config)))

	// check that it ends up on a different node than c3
	assert.NotEqual(t, c4node.ID, c3node.ID, "")

	// nodes: 2G 0CPU, 1G 0CPU, 2G 0CPU

	// add another container 1G
	config = createConfig(1, 0)
	c5node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, c5node.AddContainer(createContainer("c5", config)))

	// check that it ends up on a different node than c4 and c3
	assert.NotEqual(t, c5node.ID, c4node.ID, "")
	assert.NotEqual(t, c5node.ID, c3node.ID, "")

	// nodes: 2G 0CPU, 2G 0CPU, 2G 0CPU

	// try to add another container
	config = createConfig(3, 0)
	_, err = s.PlaceContainer(config, nodes)
	assert.Error(t, err)

	// clear a node
	c1node.CleanupContainers()

	// nodes: 0G 0CPU, 2G 0CPU, 2G 0CPU

	// add another container
	config = createConfig(4, 0)
	c7node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, c7node.AddContainer(createContainer("c7", config)))

	// check it ends up on the node we just cleared
	assert.Equal(t, c7node.ID, c1node.ID, "")
	assert.Equal(t, len(c7node.Containers()), 1, "")

	// nodes: 4G 0CPU, 2G 0CPU, 2G 0CPU

	// add a node with 1G & 1CPU
	config = createConfig(1, 1)
	c8node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, c8node.AddContainer(createContainer("c8", config)))

	// check that it ends up on a different node than c7
	assert.NotEqual(t, c8node.ID, c7node.ID, "")

	// nodes: 4G 0CPU, 3G 1CPU, 2G 0CPU

	// add a node with 1G & 1CPU
	config = createConfig(1, 1)
	c9node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, c9node.AddContainer(createContainer("c9", config)))

	// check that it ends up on a different node than c7 and c8
	assert.NotEqual(t, c9node.ID, c8node.ID, "")
	assert.NotEqual(t, c9node.ID, c7node.ID, "")

	// nodes: 4G 0CPU, 3G 1CPU, 3G 1CPU

	// add a node with 0G & 1CPU
	config = createConfig(0, 1)
	c10node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, c10node.AddContainer(createContainer("c10", config)))

	// check that it ends up on the same node as c7
	assert.Equal(t, c10node.ID, c7node.ID, "")

	// nodes: 4G 1CPU, 3G 1CPU, 3G 1CPU
}
