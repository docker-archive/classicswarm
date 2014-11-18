package filter

import (
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

type Filter interface {
	// Return a subset of nodes that were accepted by the filtering policy.
	Filter(*dockerclient.ContainerConfig, []*cluster.Node) ([]*cluster.Node, error)
}

// Apply a set of filters in batch.
func ApplyFilters(filters []Filter, config *dockerclient.ContainerConfig, nodes []*cluster.Node) ([]*cluster.Node, error) {
	var err error

	for _, filter := range filters {
		nodes, err = filter.Filter(config, nodes)
		if err != nil {
			return nil, err
		}
	}
	return nodes, nil
}
