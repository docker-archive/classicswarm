package strategy

import (
	"fmt"
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/stretchr/testify/assert"
)

func TestPlaceContainerMemory(t *testing.T) {
	s := &BinPackingPlacementStrategy{}

	nodes := []*cluster.Node{}
	for i := 0; i < 2; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 2, 1))
	}

	// add 1 container 1G
	config := createConfig(1, 0)
	node1, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node1.AddContainer(createContainer("c1", config)))

	assert.Equal(t, node1.ReservedMemory(), 1024*1024*1024)

	// add another container 1G
	config = createConfig(1, 1)
	node2, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node2.AddContainer(createContainer("c2", config)))

	assert.Equal(t, node2.ReservedMemory(), int64(2*1024*1024*1024))

	// check that both containers ended on the same node
	assert.Equal(t, node1.ID, node2.ID, "")
	assert.Equal(t, len(node1.Containers()), len(node2.Containers()), "")
}

func TestPlaceContainerCPU(t *testing.T) {
	s := &BinPackingPlacementStrategy{}

	nodes := []*cluster.Node{}
	for i := 0; i < 2; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 1, 2))
	}

	// add 1 container 1CPU
	config := createConfig(0, 1)
	node1, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node1.AddContainer(createContainer("c1", config)))

	assert.Equal(t, node1.ReservedCpus(), 1)

	// add another container 1CPU
	config = createConfig(0, 1)
	node2, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node2.AddContainer(createContainer("c2", config)))
	assert.Equal(t, node2.ReservedCpus(), 2)

	// check that both containers ended on the same node
	assert.Equal(t, node1.ID, node2.ID, "")
	assert.Equal(t, len(node1.Containers()), len(node2.Containers()), "")
}

func TestPlaceContainerHuge(t *testing.T) {
	s := &BinPackingPlacementStrategy{}

	nodes := []*cluster.Node{}
	for i := 0; i < 100; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 1, 1))
	}

	// add 100 container 1CPU
	for i := 0; i < 100; i++ {
		node, err := s.PlaceContainer(createConfig(0, 1), nodes)
		assert.NoError(t, err)
		assert.NoError(t, node.AddContainer(createContainer(fmt.Sprintf("c%d", i), createConfig(0, 100))))
	}

	// try to add another container 1CPU
	_, err := s.PlaceContainer(createConfig(0, 1), nodes)
	assert.Error(t, err)

	// add 100 container 1G
	for i := 100; i < 200; i++ {
		node, err := s.PlaceContainer(createConfig(1, 0), nodes)
		assert.NoError(t, err)
		assert.NoError(t, node.AddContainer(createContainer(fmt.Sprintf("c%d", i), createConfig(1, 0))))
	}

	// try to add another container 1G
	_, err = s.PlaceContainer(createConfig(1, 0), nodes)
	assert.Error(t, err)
}

func TestPlaceContainerOvercommit(t *testing.T) {
	s, err := New("binpacking")
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
func TestPlaceContainerDemo(t *testing.T) {
	s := &BinPackingPlacementStrategy{}

	nodes := []*cluster.Node{}
	for i := 0; i < 3; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 2, 4))
	}

	// try to place a 10G container
	config := createConfig(10, 0)
	_, err := s.PlaceContainer(config, nodes)

	// check that it refuses because the cluster is full
	assert.Error(t, err)

	// add one container 1G
	config = createConfig(1, 0)
	node1, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node1.AddContainer(createContainer("c1", config)))

	// add another container 1G
	config = createConfig(1, 0)
	node1bis, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node1bis.AddContainer(createContainer("c2", config)))

	// check that both containers ended on the same node
	assert.Equal(t, node1.ID, node1bis.ID, "")
	assert.Equal(t, len(node1.Containers()), len(node1bis.Containers()), "")

	// add another container 2G
	config = createConfig(2, 0)
	node2, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node2.AddContainer(createContainer("c3", config)))

	// check that it ends up on another node
	assert.NotEqual(t, node1.ID, node2.ID, "")

	// add another container 1G
	config = createConfig(1, 0)
	node3, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node3.AddContainer(createContainer("c4", config)))

	// check that it ends up on another node
	assert.NotEqual(t, node1.ID, node3.ID, "")
	assert.NotEqual(t, node2.ID, node3.ID, "")

	// add another container 1G
	config = createConfig(1, 0)
	node3bis, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node3bis.AddContainer(createContainer("c5", config)))

	// check that it ends up on the same node
	assert.Equal(t, node3.ID, node3bis.ID, "")

	// try to add another container
	config = createConfig(1, 0)
	_, err = s.PlaceContainer(config, nodes)

	// check that it refuses because the cluster is full
	assert.Error(t, err)

	// remove container in the middle
	node2.CleanupContainers()

	// add another container
	config = createConfig(1, 0)
	node2bis, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node2bis.AddContainer(createContainer("c6", config)))

	// check it ends up on `node3`
	assert.Equal(t, node2.ID, node2bis.ID, "")
	assert.Equal(t, len(node2.Containers()), len(node2bis.Containers()), "")
}

func TestComplexPlacement(t *testing.T) {
	s := &BinPackingPlacementStrategy{}

	nodes := []*cluster.Node{}
	for i := 0; i < 2; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 4, 4))
	}

	// add one container 2G
	config := createConfig(2, 0)
	node1, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node1.AddContainer(createContainer("c1", config)))

	// add one container 3G
	config = createConfig(3, 0)
	node2, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node2.AddContainer(createContainer("c2", config)))

	// check that they end up on separate nodes
	assert.NotEqual(t, node1.ID, node2.ID)

	// add one container 1G
	config = createConfig(1, 0)
	node3, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node3.AddContainer(createContainer("c3", config)))

	// check that it ends up on the same node as the 3G
	assert.Equal(t, node2.ID, node3.ID)
}
