package strategy

import (
	"github.com/docker/libcluster"
	"github.com/samalba/dockerclient"
)

type PlacementStrategy interface {
	// Given a container configuration and a set of nodes, select the target
	// node where the container should be scheduled.
	PlaceContainer(config *dockerclient.ContainerConfig, nodes []*libcluster.Node) (*libcluster.Node, error)
}
