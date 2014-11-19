package strategy

import (
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestPlaceContainer(t *testing.T) {
	var (
		s = &BinPackingPlacementStrategy{}

		nodes = []*cluster.Node{
			cluster.NewNode("node-1", ""),
			cluster.NewNode("node-2", ""),
			cluster.NewNode("node-3", ""),
		}

		config1 = &dockerclient.ContainerConfig{Memory: 1024 * 1024 * 1024, CpuShares: 1}
		config2 = &dockerclient.ContainerConfig{Memory: 2 * 1024 * 1024 * 1024, CpuShares: 1}

		container1 = &cluster.Container{Container: dockerclient.Container{Id: "c1"},
			Info: dockerclient.ContainerInfo{Config: config1}}
		container2 = &cluster.Container{Container: dockerclient.Container{Id: "c2"},
			Info: dockerclient.ContainerInfo{Config: config1}}
		container3 = &cluster.Container{Container: dockerclient.Container{Id: "c3"},
			Info: dockerclient.ContainerInfo{Config: config2}}
		container4 = &cluster.Container{Container: dockerclient.Container{Id: "c4"},
			Info: dockerclient.ContainerInfo{Config: config1}}
		container5 = &cluster.Container{Container: dockerclient.Container{Id: "c5"},
			Info: dockerclient.ContainerInfo{Config: config1}}
		container6 = &cluster.Container{Container: dockerclient.Container{Id: "c6"},
			Info: dockerclient.ContainerInfo{Config: config1}}
	)

	for _, node := range nodes {
		node.Memory = 2 * 1024 * 1024 * 1024
		node.Cpus = 4
	}

	// add one container 1G
	node1, err := s.PlaceContainer(config1, nodes)
	assert.NoError(t, err)
	node1.AddContainer(container1)

	// add another container 1G
	node2, err := s.PlaceContainer(config1, nodes)
	assert.NoError(t, err)
	node2.AddContainer(container2)

	// check that both containers ended on the same node
	assert.Equal(t, node1.ID, node2.ID, "")
	assert.Equal(t, len(node1.Containers()), len(node2.Containers()), "")

	// add another container 2G
	node3, err := s.PlaceContainer(config2, nodes)
	assert.NoError(t, err)
	node3.AddContainer(container3)

	// check that it ends up on another node
	assert.NotEqual(t, node1.ID, node3.ID, "")

	// add another container 1G
	node4, err := s.PlaceContainer(config1, nodes)
	assert.NoError(t, err)
	node4.AddContainer(container4)

	// check that it ends up on another node
	assert.NotEqual(t, node1.ID, node4.ID, "")
	assert.NotEqual(t, node3.ID, node4.ID, "")

	// add another container 1G
	node5, err := s.PlaceContainer(config1, nodes)
	assert.NoError(t, err)
	node5.AddContainer(container5)

	// check that it ends up on the same node
	assert.Equal(t, node4.ID, node5.ID, "")

	// try to add another container
	_, err = s.PlaceContainer(config1, nodes)

	// check that it refuses because the cluster is full
	assert.Error(t, err)

	// remove container in the middle
	node3.CleanupContainers()

	// add another container
	node6, err := s.PlaceContainer(config1, nodes)
	assert.NoError(t, err)
	node6.AddContainer(container6)

	// check it ends up on `node3`
	assert.Equal(t, node3.ID, node6.ID, "")
	assert.Equal(t, len(node3.Containers()), len(node6.Containers()), "")
}
