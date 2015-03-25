package strategy

import (
	"errors"
	"math/rand"
	"time"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

// RandomPlacementStrategy randomly places the container into the cluster.
type RandomPlacementStrategy struct{}

// Initialize is exported
func (p *RandomPlacementStrategy) Initialize() error {
	rand.Seed(time.Now().UTC().UnixNano())
	return nil
}

// PlaceContainer is exported
func (p *RandomPlacementStrategy) PlaceContainer(config *dockerclient.ContainerConfig, nodes []cluster.Node) (cluster.Node, error) {
	if size := len(nodes); size > 0 {
		n := rand.Intn(len(nodes))
		for i, node := range nodes {
			if i == n {
				return node, nil
			}
		}
	}
	return nil, errors.New("No nodes running in the cluster")
}
