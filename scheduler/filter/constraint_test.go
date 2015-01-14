package filter

import (
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func testFixtures() (nodes []*cluster.Node) {
	nodes = []*cluster.Node{
		cluster.NewNode("node-0", 0),
		cluster.NewNode("node-1", 0),
		cluster.NewNode("node-2", 0),
	}
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
	return
}

func TestConstrainteFilter(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []*cluster.Node
		err    error
	)

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

func TestConstraintNotExpr(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []*cluster.Node
		err    error
	)

	// Check not (!) expression
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"constraint:name=!node0"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Check not does_not_exist. All should be found
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"constraint:name=!does_not_exist"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Check name must not start with n
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"constraint:name=!n*"},
	}, nodes)
	assert.Error(t, err)
	assert.Len(t, result, 0)

	// Check not with globber pattern
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"constraint:region=!us*"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0].Labels["region"], "eu")
}

func TestConstraintRegExp(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []*cluster.Node
		err    error
	)

	// Check with regular expression /node\d/ matches node{0..2}
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{`constraint:name=/node\d/`},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Check with regular expression /node\d/ matches node{0..2}
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{`constraint:name=/node[12]/`},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Check with regular expression ! and regexp /node[12]/ matches node[0]
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{`constraint:name=!/node[12]/`},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Validate node pinning by ! and regexp.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"constraint:node=!/node-[01]-id/"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[2])
}

func TestFilterRegExpWithEscape(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []*cluster.Node
		err    error
	)

	// Prepare node with a strange name
	node3 := cluster.NewNode("node-3", 0)
	node3.ID = "node-3-id"
	node3.Name = "node-3-name"
	node3.Labels = map[string]string{
		"name":   "foo[bar]",
		"group":  "2",
		"region": "eu",
	}
	nodes = append(nodes, node3)

	// Test filter with a strange name
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{`constraint:name=/foo\[bar\]/`},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[3])

	// Test ! filter with a strange name
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{`constraint:name=!/foo\[bar\]/`},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestFilterRegExpCaseInsensitive(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []*cluster.Node
		err    error
	)

	// Prepare node with a strange name
	node3 := cluster.NewNode("node-3", 0)
	node3.ID = "node-3-id"
	node3.Name = "node-3-name"
	node3.Labels = map[string]string{
		"name":   "aBcDeF",
		"group":  "2",
		"region": "eu",
	}
	nodes = append(nodes, node3)

	// Case-sensitive, so not match
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{`constraint:name=/abcdef/`},
	}, nodes)
	assert.Error(t, err)
	assert.Len(t, result, 0)

	// Match with case-insensitive
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{`constraint:name=/(?i)abcdef/`},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[3])
	assert.Equal(t, result[0].Labels["name"], "aBcDeF")

	// Test ! filter combined with case insensitive
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{`constraint:name=!/(?i)abc*/`},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 3)
}
