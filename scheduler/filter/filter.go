package filter

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

// Filter is exported
type Filter interface {
	Name() string

	// Return a subset of nodes that were accepted by the filtering policy.
	Filter(*dockerclient.ContainerConfig, []cluster.Node) ([]cluster.Node, error)
}

var (
	filters []Filter
	// ErrNotSupported is exported
	ErrNotSupported = errors.New("filter not supported")
)

func init() {
	filters = []Filter{
		&AffinityFilter{},
		&HealthFilter{},
		&ConstraintFilter{},
		&PortFilter{},
		&DependencyFilter{},
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
func ApplyFilters(filters []Filter, config *dockerclient.ContainerConfig, nodes []cluster.Node) ([]cluster.Node, error) {
	var err error

	for _, filter := range filters {
		nodes, err = filter.Filter(config, nodes)
		if err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func List() []string {
	names := []string{}

	for _, filter := range filters {
		names = append(names, filter.Name())
	}

	return names
}
