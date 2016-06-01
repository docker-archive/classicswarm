package filter

import (
	"errors"
	"strconv"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

var (
	// ErrNoNodeWithFreeSlotsAvailable is exported
	ErrNoNodeWithFreeSlotsAvailable = errors.New("No node with enough open slots available in the cluster")
)

//SlotsFilter only schedules containers with open slots.
type SlotsFilter struct {
}

// Name returns the name of the filter
func (f *SlotsFilter) Name() string {
	return "containerslots"
}

// Filter is exported
func (f *SlotsFilter) Filter(_ *cluster.ContainerConfig, nodes []*node.Node, _ bool) ([]*node.Node, error) {
	result := []*node.Node{}

	for _, node := range nodes {

		if slotsString, ok := node.Labels["containerslots"]; ok {
			slots, err := strconv.Atoi(slotsString) //if err => cannot cast to int, so ignore the label
			if err != nil || len(node.Containers) < slots {
				result = append(result, node)
			}
		} else {
			//no limit if there is no containerslots label
			result = append(result, node)
		}
	}

	if len(result) == 0 {
		return nil, ErrNoNodeWithFreeSlotsAvailable
	}

	return result, nil
}

// GetFilters returns just the info that this node failed, because there where no free slots
func (f *SlotsFilter) GetFilters(config *cluster.ContainerConfig) ([]string, error) {
	return []string{"available container slots"}, nil
}
