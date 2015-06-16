package strategy

import (
	"errors"
	"math/rand"
	"time"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// RandomPlacementStrategy randomly places the container into the cluster.
type RandomPlacementStrategy struct {
	r *rand.Rand
}

// Initialize is exported
func (p *RandomPlacementStrategy) Initialize() error {
	p.r = rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	return nil
}

// Name returns the name of the strategy
func (p *RandomPlacementStrategy) Name() string {
	return "random"
}

// PlaceContainer is exported
func (p *RandomPlacementStrategy) PlaceContainer(config *cluster.ContainerConfig, nodes []*node.Node) (*node.Node, error) {
	if size := len(nodes); size > 0 {
		return nodes[p.r.Intn(size)], nil
	}

	return nil, errors.New("No nodes running in the cluster")
}
