package strategy

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

type PlacementStrategy interface {
	Initialize(overcommitRatio int64) error
	// Given a container configuration and a set of nodes, select the target
	// node where the container should be scheduled.
	PlaceContainer(config *dockerclient.ContainerConfig, nodes []*cluster.Node) (*cluster.Node, error)
}

var (
	strategies      map[string]PlacementStrategy
	ErrNotSupported = errors.New("strategy not supported")
)

func init() {
	strategies = map[string]PlacementStrategy{
		"binpacking": &BinPackingPlacementStrategy{},
		"random":     &RandomPlacementStrategy{},
	}
}

func New(name string, overcommitRatio int64) (PlacementStrategy, error) {
	if strategy, exists := strategies[name]; exists {
		log.Debugf("Initializing %q strategy", name)
		err := strategy.Initialize(overcommitRatio)
		return strategy, err
	}

	return nil, ErrNotSupported
}
