package strategy

import (
	"sort"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

type BinpackPlacementStrategy struct {
}

func (p *BinpackPlacementStrategy) Initialize() error {
	return nil
}

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
