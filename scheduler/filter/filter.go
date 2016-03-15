package filter

import (
	"errors"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// Filter is exported
type Filter interface {
	Name() string

	// Return a subset of nodes that were accepted by the filtering policy.
	Filter(*cluster.ContainerConfig, []*node.Node, bool) ([]*node.Node, error)

	// Return a list of constraints/filters provided
	GetFilters(*cluster.ContainerConfig) ([]string, error)
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
			// special case for when no healthy nodes are found
			if filter.Name() == "health" {
				return nil, err
			}
			return nil, fmt.Errorf("Unable to find a node that satisfies the following conditions %s", listAllFilters(filters, config, filter.Name()))
		}
	}
	return candidates, nil
}

// listAllFilters creates a string containing all applied filters
func listAllFilters(filters []Filter, config *cluster.ContainerConfig, lastFilter string) string {
	allFilters := ""
	for _, filter := range filters {
		list, err := filter.GetFilters(config)
		if err == nil && len(list) > 0 {
			allFilters = fmt.Sprintf("%s\n%v", allFilters, list)
		}
		if filter.Name() == lastFilter {
			return allFilters
		}
	}
	return allFilters
}

// List returns the names of all the available filters
func List() []string {
	names := []string{}

	for _, filter := range filters {
		names = append(names, filter.Name())
	}

	return names
}
