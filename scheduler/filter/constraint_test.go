package filter

import (
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestConstrainteFilter(t *testing.T) {
	var (
		f     = ConstraintFilter{}
		nodes = []*cluster.Node{
			cluster.NewNode("node-0"),
			cluster.NewNode("node-1"),
			cluster.NewNode("node-2"),
		}
		result []*cluster.Node
		err    error
	)

	nodes[0].ID = "node-0-id"
	nodes[0].Name = "node-0-name"
	nodes[0].Labels = map[string]string{
		"name":   "node0",
		"group":  "1",
		"region": "us-west",
	}

	nodes[1].ID = "node-1-id"
	nodes[1].Name = "node-1-name"
	nodes[1].Labels = map[string]string{
		"name":   "node1",
		"group":  "1",
		"region": "us-east",
	}

	nodes[2].ID = "node-2-id"
	nodes[2].Name = "node-2-name"
	nodes[2].Labels = map[string]string{
		"name":   "node2",
		"group":  "2",
		"region": "eu",
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

	// Validate node pinning by id.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"constraint:node=node-2-id"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[2])

	// Validate node pinning by name.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"constraint:node=node-1-name"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// Make sure constraints are evaluated as logical ANDs.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"constraint:name=node0", "constraint:group=1"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Check matching
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"constraint:region=us"},
	}, nodes)
	assert.Error(t, err)
	assert.Len(t, result, 0)

	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"constraint:region=us*"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}
