package strategy

import (
	"sort"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// MemoryPlacementStrategy places a container on the node with the fewest running containers.
type MemoryPlacementStrategy struct {
}

// Initialize a MemoryPlacementStrategy.
func (p *MemoryPlacementStrategy) Initialize() error {
	return nil
}

// Name returns the name of the strategy.
func (p *MemoryPlacementStrategy) Name() string {
	return "memory"
}

// RankAndSort sorts nodes based on the spread strategy applied to the container config.
func (p *MemoryPlacementStrategy) RankAndSort(config *cluster.ContainerConfig, nodes []*node.Node) ([]*node.Node, error) {
	const cpuFactor int64 = 0
	const memoryFactor int64 = -10
	weightedNodes, err := weighNodesByResources(config, nodes, cpuFactor, memoryFactor)
	if err != nil {
		return nil, err
	}

	sort.Sort(weightedNodes)
	output := make([]*node.Node, len(weightedNodes))
	for i, n := range weightedNodes {
		output[i] = n.Node
	}
	return output, nil
}
