package filter

import (
	"errors"
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
	"github.com/stretchr/testify/assert"
)

func testFixturesAllHealthyNode() []*node.Node {
	return []*node.Node{
		{
			ID:        "node-0-id",
			Name:      "node-0-name",
			IsHealthy: true,
		},

		{
			ID:        "node-1-id",
			Name:      "node-1-name",
			IsHealthy: true,
		},
	}
}

func testFixturesPartHealthyNode() []*node.Node {
	return []*node.Node{
		{
			ID:        "node-0-id",
			Name:      "node-0-name",
			IsHealthy: false,
		},

		{
			ID:        "node-1-id",
			Name:      "node-1-name",
			IsHealthy: true,
		},
	}
}

func testFixturesNoHealthyNode() []*node.Node {
	return []*node.Node{
		{
			ID:        "node-0-id",
			Name:      "node-0-name",
			IsHealthy: false,
		},

		{
			ID:        "node-1-id",
			Name:      "node-1-name",
			IsHealthy: false,
		},
	}
}

func TestHealthyFilter(t *testing.T) {
	var (
		f                         = HealthFilter{}
		nodesAllHealth            = testFixturesAllHealthyNode()
		nodesPartHealth           = testFixturesPartHealthyNode()
		nodesNoHealth             = testFixturesNoHealthyNode()
		errNoHealthyNodeAvailable = errors.New("No healthy node available in the cluster")
		result                    []*node.Node
		err                       error
	)

	result, err = f.Filter(&cluster.ContainerConfig{}, nodesAllHealth)
	assert.NoError(t, err)
	assert.Equal(t, result, nodesAllHealth)

	result, err = f.Filter(&cluster.ContainerConfig{}, nodesPartHealth)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodesPartHealth[1])

	result, err = f.Filter(&cluster.ContainerConfig{}, nodesNoHealth)
	assert.Equal(t, err, errNoHealthyNodeAvailable)
	assert.Nil(t, result)
}
