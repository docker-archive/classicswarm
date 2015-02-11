package options

import (
	"crypto/tls"

	"github.com/docker/swarm/filter"
	"github.com/docker/swarm/state"
	"github.com/docker/swarm/strategy"
)

type SchedulerOptions struct {
	Strategy strategy.PlacementStrategy
	Filters  []filter.Filter

	Store           *state.Store
	TLSConfig       *tls.Config
	OvercommitRatio float64
}
