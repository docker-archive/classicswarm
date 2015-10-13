package strategy

import (
	"math/rand"
	"time"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// RandomPlacementStrategy randomly places the container into the cluster.
type RandomPlacementStrategy struct {
	r *rand.Rand
}

// Initialize a RandomPlacementStrategy.
func (p *RandomPlacementStrategy) Initialize() error {
	p.r = rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	return nil
}

// Name returns the name of the strategy.
func (p *RandomPlacementStrategy) Name() string {
	return "random"
}

// RankAndSort randomly sorts the list of nodes.
func (p *RandomPlacementStrategy) RankAndSort(config *cluster.ContainerConfig, nodes []*node.Node) ([]*node.Node, error) {
	for i := len(nodes) - 1; i > 0; i-- {
		j := p.r.Intn(i + 1)
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
	return nodes, nil
}
