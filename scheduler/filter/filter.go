package filter

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// Filter is exported
type Filter interface {
	Name() string

	// Return a subset of nodes that were accepted by the filtering policy.
	Filter(*cluster.ContainerConfig, []*node.Node, bool) ([]*node.Node, error)
}

var (
	filters []Filter
	// ErrNotSupported is exported
	ErrNotSupported = errors.New("filter not supported")
)

func init() {
	filters = []Filter{
		&HealthFilter{},
		&PortFilter{},
		&DependencyFilter{},
		&AffinityFilter{},
		&ConstraintFilter{},
	}
}

// New is exported
func New(names []string) ([]Filter, error) {
	var selectedFilters []Filter

	for _, name := range names {
		found := false
		for _, filter := range filters {
			if filter.Name() == name {
				log.WithField("name", name).Debug("Initializing filter")
				selectedFilters = append(selectedFilters, filter)
				found = true
				break
			}
		}
		if !found {
			return nil, ErrNotSupported
		}
	}
	return selectedFilters, nil
}

// ApplyFilters applies a set of filters in batch.
func ApplyFilters(filters []Filter, config *cluster.ContainerConfig, nodes []*node.Node, soft bool) ([]*node.Node, error) {
	var (
		err        error
		candidates = nodes
	)

	for _, filter := range filters {
		candidates, err = filter.Filter(config, candidates, soft)
		if err != nil {
			return nil, err
		}
	}
	return candidates, nil
}

// List returns the names of all the available filters
func List() []string {
	names := []string{}

	for _, filter := range filters {
		names = append(names, filter.Name())
	}

	return names
}
