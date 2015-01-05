package strategy

import (
	"errors"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

type PlacementStrategy interface {
	Initialize(string) error
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

func New(nameAndOpts string) (PlacementStrategy, error) {
	var (
		parts = strings.SplitN(nameAndOpts, ":", 2)
		name  = parts[0]
		opts  string
	)
	if len(parts) == 2 {
		opts = parts[1]
	}

	if strategy, exists := strategies[name]; exists {
		log.Debugf("Initializing %q strategy with %q", name, opts)
		err := strategy.Initialize(opts)
		return strategy, err
	}

	return nil, ErrNotSupported
}
