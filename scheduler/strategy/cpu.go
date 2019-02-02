package strategy

import (
	"sort"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// CpuPlacementStrategy places a container on the node with the fewest running containers.
type CpuPlacementStrategy struct {
}

// Initialize a CpuPlacementStrategy.
func (p *CpuPlacementStrategy) Initialize() error {
	return nil
}

// Name returns the name of the strategy.
func (p *CpuPlacementStrategy) Name() string {
	return "cpu"
}

// RankAndSort sorts nodes based on the spread strategy applied to the container config.
func (p *CpuPlacementStrategy) RankAndSort(config *cluster.ContainerConfig, nodes []*node.Node) ([]*node.Node, error) {
	const cpuFactor int64 = -10
	const memoryFactor int64 = 0
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
