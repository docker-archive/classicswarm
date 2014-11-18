package filter

import (
	"github.com/docker/libcluster/swarm"
	"github.com/samalba/dockerclient"
)

type Filter interface {
	// Return a subset of nodes that were accepted by the filtering policy.
	Filter(*dockerclient.ContainerConfig, []*swarm.Node) ([]*swarm.Node, error)
}

// Apply a set of filters in batch.
func ApplyFilters(filters []Filter, config *dockerclient.ContainerConfig, nodes []*swarm.Node) ([]*swarm.Node, error) {
	var err error

	for _, filter := range filters {
		nodes, err = filter.Filter(config, nodes)
		if err != nil {
			return nil, err
		}
	}
	return nodes, nil
}
