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

	// sort by highest weight
	sort.Sort(weightedNodes)

	return weightedNodes[0].Node, nil
}
