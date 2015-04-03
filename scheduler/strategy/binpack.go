package strategy

import (
	"sort"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

// BinpackPlacementStrategy is exported
type BinpackPlacementStrategy struct {
}

// Initialize is exported
func (p *BinpackPlacementStrategy) Initialize() error {
	return nil
}

// Name returns the name of the strategy
func (p *BinpackPlacementStrategy) Name() string {
	return "binpack"
}

// PlaceContainer is exported
func (p *BinpackPlacementStrategy) PlaceContainer(config *dockerclient.ContainerConfig, nodes []cluster.Node) (cluster.Node, error) {
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
		if len(node.Node.Containers()) > len(topNode.Node.Containers()) {
			topNode = node
		}
	}

	return topNode.Node, nil
}
