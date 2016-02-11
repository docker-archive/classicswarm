package filter

import (
	"errors"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

var (
	// ErrNoHealthyNodeAvailable is exported
	ErrNoHealthyNodeAvailable = errors.New("No healthy node available in the cluster")
)

// HealthFilter only schedules containers on healthy nodes.
type HealthFilter struct {
}

// Name returns the name of the filter
func (f *HealthFilter) Name() string {
	return "health"
}

// Filter is exported
func (f *HealthFilter) Filter(_ *cluster.ContainerConfig, nodes []*node.Node, _ bool) ([]*node.Node, error) {
	result := []*node.Node{}
	for _, node := range nodes {
		if node.IsHealthy() {
			result = append(result, node)
		}
	}

	if len(result) == 0 {
		return nil, ErrNoHealthyNodeAvailable
	}

	return result, nil
}

// GetFilters returns
func (f *HealthFilter) GetFilters(config *cluster.ContainerConfig) ([]string, error) {
	return nil, nil
}
