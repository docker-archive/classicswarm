package strategy

import (
	"errors"

	"github.com/docker/libcluster"
	"github.com/samalba/dockerclient"
)

// Randomly place the container into the cluster.
type RandomPlacementStrategy struct {
}

func (p *RandomPlacementStrategy) PlaceContainer(config *dockerclient.ContainerConfig, nodes []*libcluster.Node) (*libcluster.Node, error) {
	for _, node := range nodes {
		return node, nil
	}
	return nil, errors.New("No nodes running in the cluster")
}
