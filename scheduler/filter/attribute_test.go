package filter

import (
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestAttributeFilter(t *testing.T) {
	var (
		f     = AttributeFilter{}
		nodes = []*cluster.Node{
			cluster.NewNode("node-1"),
			cluster.NewNode("node-2"),
			cluster.NewNode("node-3"),
		}
		result []*cluster.Node
		err    error
	)

	nodes[0].Labels = map[string]string{
		"name":  "node0",
		"group": "1",
	}

	nodes[1].Labels = map[string]string{
		"name":  "node1",
		"group": "1",
	}

	nodes[2].Labels = map[string]string{
		"name":  "node2",
		"group": "2",
	}

	// Without constraints we should get the unfiltered list of nodes back.
	result, err = f.Filter(&dockerclient.ContainerConfig{}, nodes)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)

	// Set a constraint that cannot be fullfilled and expect an error back.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"constraint:does_not_exist=true"},
	}, nodes)
	assert.Error(t, err)

	// Set a contraint that can only be filled by a single node.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"constraint:name=node1"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// This constraint can only be fullfilled by a subset of nodes.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"constraint:group=1"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[2])
}
