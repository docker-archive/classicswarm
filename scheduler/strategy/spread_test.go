package strategy

import (
	"fmt"
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/stretchr/testify/assert"
)

func TestSpreadPlaceEqualWeight(t *testing.T) {
	s := &SpreadPlacementStrategy{}

	nodes := []cluster.Node{}
	for i := 0; i < 2; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 4, 0))
	}

	// add 1 container 2G on node1
	config := createConfig(2, 0)
	assert.NoError(t, AddContainer(nodes[0], createContainer("c1", config)))
	assert.Equal(t, nodes[0].UsedMemory(), 2*1024*1024*1024)

	// add 2 containers 1G on node2
	config = createConfig(1, 0)
	assert.NoError(t, AddContainer(nodes[1], createContainer("c2", config)))
	assert.NoError(t, AddContainer(nodes[1], createContainer("c3", config)))
	assert.Equal(t, nodes[1].UsedMemory(), int64(2*1024*1024*1024))

	// add another container 1G
	config = createConfig(1, 0)
	node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, AddContainer(node, createContainer("c4", config)))
	assert.Equal(t, node.UsedMemory(), 3*1024*1024*1024)

	// check that the last container ended on the node with the lowest number of containers
	assert.Equal(t, node.ID(), nodes[0].ID())
	assert.Equal(t, len(nodes[0].Containers()), len(nodes[1].Containers()))

}

func TestSpreadPlaceContainerMemory(t *testing.T) {
	s := &SpreadPlacementStrategy{}

	nodes := []cluster.Node{}
	for i := 0; i < 2; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 2, 0))
	}

	// add 1 container 1G
	config := createConfig(1, 0)
	node1, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, AddContainer(node1, createContainer("c1", config)))
	assert.Equal(t, node1.UsedMemory(), 1024*1024*1024)

	// add another container 1G
	config = createConfig(1, 0)
	node2, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, AddContainer(node2, createContainer("c2", config)))
	assert.Equal(t, node2.UsedMemory(), int64(1024*1024*1024))

	// check that both containers ended on different node
	assert.NotEqual(t, node1.ID(), node2.ID())
	assert.Equal(t, len(node1.Containers()), len(node2.Containers()), "")
}

func TestSpreadPlaceContainerCPU(t *testing.T) {
	s := &SpreadPlacementStrategy{}

	nodes := []cluster.Node{}
	for i := 0; i < 2; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 0, 2))
	}

	// add 1 container 1CPU
	config := createConfig(0, 1)
	node1, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, AddContainer(node1, createContainer("c1", config)))
	assert.Equal(t, node1.UsedCpus(), 1)

	// add another container 1CPU
	config = createConfig(0, 1)
	node2, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, AddContainer(node2, createContainer("c2", config)))
	assert.Equal(t, node2.UsedCpus(), 1)

	// check that both containers ended on different node
	assert.NotEqual(t, node1.ID(), node2.ID())
	assert.Equal(t, len(node1.Containers()), len(node2.Containers()), "")
}

func TestSpreadPlaceContainerHuge(t *testing.T) {
	s := &SpreadPlacementStrategy{}

	nodes := []cluster.Node{}
	for i := 0; i < 100; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 1, 1))
	}

	// add 100 container 1CPU
	for i := 0; i < 100; i++ {
		node, err := s.PlaceContainer(createConfig(0, 1), nodes)
		assert.NoError(t, err)
		assert.NoError(t, AddContainer(node, createContainer(fmt.Sprintf("c%d", i), createConfig(0, 1))))
	}

	// try to add another container 1CPU
	_, err := s.PlaceContainer(createConfig(0, 1), nodes)
	assert.Error(t, err)

	// add 100 container 1G
	for i := 100; i < 200; i++ {
		node, err := s.PlaceContainer(createConfig(1, 0), nodes)
		assert.NoError(t, err)
		assert.NoError(t, AddContainer(node, createContainer(fmt.Sprintf("c%d", i), createConfig(1, 0))))
	}

	// try to add another container 1G
	_, err = s.PlaceContainer(createConfig(1, 0), nodes)
	assert.Error(t, err)
}

func TestSpreadPlaceContainerOvercommit(t *testing.T) {
	s := &SpreadPlacementStrategy{}

	nodes := []cluster.Node{createNode("node-1", 100, 1)}

	config := createConfig(0, 0)

	// Below limit should still work.
	config.Memory = 90 * 1024 * 1024 * 1024
	node, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.Equal(t, node, nodes[0])

	// At memory limit should still work.
	config.Memory = 100 * 1024 * 1024 * 1024
	node, err = s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.Equal(t, node, nodes[0])

	// Up to 105% it should still work.
	config.Memory = 105 * 1024 * 1024 * 1024
	node, err = s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.Equal(t, node, nodes[0])

	// Above it should return an error.
	config.Memory = 106 * 1024 * 1024 * 1024
	node, err = s.PlaceContainer(config, nodes)
	assert.Error(t, err)
}

func TestSpreadComplexPlacement(t *testing.T) {
	s := &SpreadPlacementStrategy{}

	nodes := []cluster.Node{}
	for i := 0; i < 2; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 4, 4))
	}

	// add one container 2G
	config := createConfig(2, 0)
	node1, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, AddContainer(node1, createContainer("c1", config)))

	// add one container 3G
	config = createConfig(3, 0)
	node2, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, AddContainer(node2, createContainer("c2", config)))

	// check that they end up on separate nodes
	assert.NotEqual(t, node1.ID(), node2.ID())

	// add one container 1G
	config = createConfig(1, 0)
	node3, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, AddContainer(node3, createContainer("c3", config)))

	// check that it ends up on the same node as the 2G
	assert.Equal(t, node1.ID(), node3.ID())
}
