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

// PlaceContainer places a container on the node with the most running containers.
func (p *BinpackPlacementStrategy) PlaceContainer(config *cluster.ContainerConfig, nodes []*node.Node) (*node.Node, error) {
	weightedNodes, err := weighNodes(config, nodes)
	if err != nil {
		return nil, err
	}

	// sort by highest weight
	sort.Sort(sort.Reverse(weightedNodes))

	topNode := weightedNodes[0]
	for _, node := range weightedNodes {
		if node.Weight != topNode.Weight {
			break
		}
		if len(node.Node.Containers) > len(topNode.Node.Containers) {
			topNode = node
		}
	}

	return topNode.Node, nil
}
