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

// PlaceContainer places a container on the node with the fewest running containers.
func (p *SpreadPlacementStrategy) PlaceContainer(config *cluster.ContainerConfig, nodes []*node.Node) (*node.Node, error) {
	weightedNodes, err := weighNodes(config, nodes)
	if err != nil {
		return nil, err
	}

	// sort by lowest weight
	sort.Sort(weightedNodes)

	bottomNode := weightedNodes[0]
	for _, node := range weightedNodes {
		if node.Weight != bottomNode.Weight {
			break
		}
		if len(node.Node.Containers) < len(bottomNode.Node.Containers) {
			bottomNode = node
		}
	}

	return bottomNode.Node, nil
}
