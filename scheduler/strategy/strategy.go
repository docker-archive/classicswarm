package strategy

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

type PlacementStrategy interface {
	Initialize() error
	// Given a container configuration and a set of nodes, select the target
	// node where the container should be scheduled.
	PlaceContainer(config *dockerclient.ContainerConfig, nodes []cluster.Node) (cluster.Node, error)
}

var (
	strategies              map[string]PlacementStrategy
	ErrNotSupported         = errors.New("strategy not supported")
	ErrNoResourcesAvailable = errors.New("no resources available to schedule container")
)

func init() {
	strategies = map[string]PlacementStrategy{
		"binpacking": &BinpackPlacementStrategy{}, //compat
		"binpack":    &BinpackPlacementStrategy{},
		"spread":     &SpreadPlacementStrategy{},
		"random":     &RandomPlacementStrategy{},
	}
}

func New(name string) (PlacementStrategy, error) {
	if strategy, exists := strategies[name]; exists {
		log.WithField("name", name).Debugf("Initializing strategy")
		err := strategy.Initialize()
		return strategy, err
	}

	return nil, ErrNotSupported
}
