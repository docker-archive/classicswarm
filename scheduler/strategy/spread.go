package strategy

import (
	"sort"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

type SpreadPlacementStrategy struct {
}

func (p *SpreadPlacementStrategy) Initialize() error {
	return nil
}

func (p *SpreadPlacementStrategy) PlaceContainer(config *dockerclient.ContainerConfig, nodes []cluster.Node) (cluster.Node, error) {
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
		if len(node.Node.Containers()) < len(bottomNode.Node.Containers()) {
			bottomNode = node
		}
	}

	return bottomNode.Node, nil
}
