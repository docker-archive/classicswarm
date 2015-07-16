package strategy

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// PlacementStrategy is the interface for a container placement strategy.
type PlacementStrategy interface {
	// Name of the strategy
	Name() string
	// Initialize performs any initial configuration required by the strategy and returns
	// an error if one is encountered.
	// If no initial configuration is needed, this may be a no-op and return a nil error.
	Initialize() error
	// Given a container configuration and a set of nodes, select the target
	// node where the container should be scheduled. PlaceContainer returns
	// an error if there is no available node on which to schedule the container.
	PlaceContainer(config *cluster.ContainerConfig, nodes []*node.Node) (*node.Node, error)
}

var (
	strategies []PlacementStrategy
	// ErrNotSupported is the error returned when a strategy name does not match
	// any supported placement strategy.
	ErrNotSupported = errors.New("strategy not supported")
	// ErrNoResourcesAvailable is the error returned when there are no resources
	// available to schedule a container. This can occur if there are no nodes in
	// the cluster or if no node contains sufficient resources for the container.
	ErrNoResourcesAvailable = errors.New("no resources available to schedule container")
)

func init() {
	strategies = []PlacementStrategy{
		&SpreadPlacementStrategy{},
		&BinpackPlacementStrategy{},
		&RandomPlacementStrategy{},
	}
}

// New creates a new PlacementStrategy for the given strategy name.
func New(name string) (PlacementStrategy, error) {
	if name == "binpacking" { //TODO: remove this compat
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

// List returns the names of all the available strategies.
func List() []string {
	names := []string{}

	for _, strategy := range strategies {
		names = append(names, strategy.Name())
	}

	return names
}
