package strategy

import (
	"sort"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// BinpackPlacementStrategy places a container onto the most packed node in the cluster.
type BinpackPlacementStrategy struct {
}

// Initialize a BinpackPlacementStrategy.
func (p *BinpackPlacementStrategy) Initialize() error {
	return nil
}

// Name returns the name of the strategy.
func (p *BinpackPlacementStrategy) Name() string {
	return "binpack"
}

// RankAndSort sorts nodes based on the binpack strategy applied to the container config.
func (p *BinpackPlacementStrategy) RankAndSort(config *cluster.ContainerConfig, nodes []*node.Node) ([]*node.Node, error) {
	weightedNodes, err := weighNodes(config, nodes)
	if err != nil {
		return nil, err
	}

	sort.Sort(sort.Reverse(weightedNodes))
	output := make([]*node.Node, len(weightedNodes))
	for i, n := range weightedNodes {
		output[i] = n.Node
	}
	return output, nil
}
