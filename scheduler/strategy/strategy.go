package strategy

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

// PlacementStrategy is exported
type PlacementStrategy interface {
	Name() string

	Initialize() error
	// Given a container configuration and a set of nodes, select the target
	// node where the container should be scheduled.
	PlaceContainer(config *dockerclient.ContainerConfig, nodes []cluster.Node) (cluster.Node, error)
}

var (
	strategies []PlacementStrategy
	// ErrNotSupported is exported
	ErrNotSupported = errors.New("strategy not supported")
	// ErrNoResourcesAvailable is exported
	ErrNoResourcesAvailable = errors.New("no resources available to schedule container")
)

func init() {
	strategies = []PlacementStrategy{
		&SpreadPlacementStrategy{},
		&BinpackPlacementStrategy{},
		&RandomPlacementStrategy{},
	}
}

// New is exported
func New(name string) (PlacementStrategy, error) {
	if name == "binpacking" { //compat
		name = "binpack"
	}

	for _, strategy := range strategies {
		if strategy.Name() == name {
			log.WithField("name", name).Debugf("Initializing strategy")
			err := strategy.Initialize()
			return strategy, err
		}
	}

	return nil, ErrNotSupported
}

func List() []string {
	names := []string{}

	for _, strategy := range strategies {
		names = append(names, strategy.Name())
	}

	return names
}
