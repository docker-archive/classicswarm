package strategy

import (
	"errors"
	"math/rand"
	"time"

	"github.com/docker/swarm/scheduler/node"
	"github.com/samalba/dockerclient"
)

// RandomPlacementStrategy randomly places the container into the cluster.
type RandomPlacementStrategy struct{}

// Initialize is exported
func (p *RandomPlacementStrategy) Initialize() error {
	rand.Seed(time.Now().UTC().UnixNano())
	return nil
}

// Name returns the name of the strategy
func (p *RandomPlacementStrategy) Name() string {
	return "random"
}

// PlaceContainer is exported
func (p *RandomPlacementStrategy) PlaceContainer(config *dockerclient.ContainerConfig, nodes []*node.Node) (*node.Node, error) {
	if size := len(nodes); size > 0 {
		return nodes[rand.Intn(size)], nil
	}

	return nil, errors.New("No nodes running in the cluster")
}
