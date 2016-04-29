package scheduler

import (
	"testing"

	containertypes "github.com/docker/engine-api/types/container"
	networktypes "github.com/docker/engine-api/types/network"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/filter"
	"github.com/docker/swarm/scheduler/node"
	"github.com/docker/swarm/scheduler/strategy"
	"github.com/stretchr/testify/assert"
)

func TestSelectNodesForContainer(t *testing.T) {
	var (
		s = Scheduler{
			strategy: &strategy.SpreadPlacementStrategy{},
			filters:  []filter.Filter{&filter.ConstraintFilter{}},
		}

		nodes = []*node.Node{
			{
				ID:          "node-0-id",
				Name:        "node-0-name",
				Addr:        "node-0",
				TotalMemory: 1 * 1024 * 1024 * 1024,
				TotalCpus:   1,
				Labels: map[string]string{
					"group": "1",
				},
			},

			{
				ID:          "node-1-id",
				Name:        "node-1-name",
				Addr:        "node-1",
				TotalMemory: 1 * 1024 * 1024 * 1024,
				TotalCpus:   2,
				Labels: map[string]string{
					"group": "2",
				},
			},
		}

		config = cluster.BuildContainerConfig(containertypes.Config{
			Env: []string{"constraint:group==~1"},
		}, containertypes.HostConfig{
			Resources: containertypes.Resources{
				Memory:    1024 * 1024 * 1024,
				CPUShares: 2,
			},
		}, networktypes.NetworkingConfig{})
	)
	candidates, err := s.SelectNodesForContainer(nodes, config)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(candidates))
	assert.Equal(t, "node-1-id", candidates[0].ID)

}
