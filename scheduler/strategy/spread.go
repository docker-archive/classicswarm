package strategy

import (
	"sort"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// SpreadPlacementStrategy places a container on the node with the fewest running containers.
type SpreadPlacementStrategy struct {
}

// Initialize a SpreadPlacementStrategy.
func (p *SpreadPlacementStrategy) Initialize() error {
	return nil
}

// Name returns the name of the strategy.
func (p *SpreadPlacementStrategy) Name() string {
	return "spread"
}

// RankAndSort sorts nodes based on the spread strategy applied to the container config.
func (p *SpreadPlacementStrategy) RankAndSort(config *cluster.ContainerConfig, nodes []*node.Node) ([]*node.Node, error) {
	// for spread, a healthy node should decrease its weight to increase its chance of being selected
	// set healthFactor to -10 to make health degree [0, 100] overpower cpu + memory (each in range [0, 100])
	const healthFactor int64 = -10
	weightedNodes, err := weighNodes(config, nodes, healthFactor)
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
