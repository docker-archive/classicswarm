package strategy

import (
	"fmt"
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/stretchr/testify/assert"
)

func TestPlaceContainerRoundRobinMemory(t *testing.T) {
	s := &RoundRobinPlacementStrategy{}

	nodes := []*cluster.Node{}
	for i := 0; i < 2; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 1, 1))
	}

	// add 1 container 1G
	config := createConfig(1, 0)
	node1, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node1.AddContainer(createContainer("c1", config)))
	assert.Equal(t, 1, len(node1.Containers()))

	// add another container 1G
	node2, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node2.AddContainer(createContainer("c2", config)))
	assert.Equal(t, 1, len(node2.Containers()))

	// check that both containers ended on different nodes
	assert.NotEqual(t, node1.ID, node2.ID, "")
	assert.Equal(t, len(node1.Containers()), len(node2.Containers()), "")
}

func TestPlaceContainerRoundRobinCPU(t *testing.T) {
	s := &RoundRobinPlacementStrategy{}

	nodes := []*cluster.Node{}
	for i := 0; i < 2; i++ {
		nodes = append(nodes, createNode(fmt.Sprintf("node-%d", i), 1, 2))
	}

	// add 1 container 1CPU
	config := createConfig(0, 1)
	node1, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node1.AddContainer(createContainer("c1", config)))

	assert.Equal(t, 1, node1.ReservedCpus())

	// add another container 1CPU
	config = createConfig(0, 1)
	node2, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node2.AddContainer(createContainer("c2", config)))
	assert.Equal(t, 1, node2.ReservedCpus())

	// check that both containers ended on different nodes
	assert.NotEqual(t, node1.ID, node2.ID, "")
	assert.Equal(t, len(node1.Containers()), len(node2.Containers()), "")
}

func TestPlaceContainerRoundRobinHuge(t *testing.T) {
	s := &RoundRobinPlacementStrategy{}

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

func TestPlaceContainerRoundRobinOvercommit(t *testing.T) {
	s, err := New("roundrobin")
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

func TestPlaceContainerRoundRobinDemo(t *testing.T) {
	s := &RoundRobinPlacementStrategy{}

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
	node2, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node2.AddContainer(createContainer("c2", config)))

	// check that both containers ended on different nodes
	assert.NotEqual(t, node1.ID, node2.ID, "")
	assert.Equal(t, len(node1.Containers()), len(node2.Containers()), "")

	// add another container 2G
	config = createConfig(2, 0)
	node3, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node3.AddContainer(createContainer("c3", config)))

	// check that it ends up on another node
	assert.NotEqual(t, node1.ID, node3.ID, "")
	assert.NotEqual(t, node2.ID, node3.ID, "")

	// add another container 1G
	config = createConfig(1, 0)
	node4, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node4.AddContainer(createContainer("c4", config)))

	// check that it ends up on node1
	assert.Equal(t, node4.ID, node1.ID, "")

	// add another container 2G
	config = createConfig(2, 0)
	_, err = s.PlaceContainer(config, nodes)

	// check that it refuses because the cluster only has capacity for 1G
	assert.Error(t, err)

	// add another container 1G
	config = createConfig(1, 0)
	node5, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node5.AddContainer(createContainer("c5", config)))

	// check that it ends up on node2
	assert.Equal(t, node5.ID, node2.ID, "")

	// remove container in the middle
	node2.CleanupContainers()

	// add another container
	config = createConfig(1, 0)
	node6, err := s.PlaceContainer(config, nodes)
	assert.NoError(t, err)
	assert.NoError(t, node6.AddContainer(createContainer("c6", config)))

	// check that it ends up on node2
	assert.Equal(t, node6.ID, node2.ID, "")
	assert.Equal(t, len(node2.Containers()), len(node6.Containers()), "")
}
