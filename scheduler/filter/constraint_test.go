package filter

import (
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func testFixtures() []*node.Node {
	return []*node.Node{
		{
			ID:   "node-0-id",
			Name: "node-0-name",
			Addr: "node-0",
			Labels: map[string]string{
				"name":   "node0",
				"group":  "1",
				"region": "us-west",
			},
		},

		{
			ID:   "node-1-id",
			Name: "node-1-name",
			Addr: "node-1",
			Labels: map[string]string{
				"name":   "node1",
				"group":  "1",
				"region": "us-east",
			},
		},

		{
			ID:   "node-2-id",
			Name: "node-2-name",
			Addr: "node-2",
			Labels: map[string]string{
				"name":   "node2",
				"group":  "2",
				"region": "eu",
			},
		},

		{
			ID:   "node-3-id",
			Name: "node-3-name",
			Addr: "node-3",
		},
	}
}

func TestConstrainteFilter(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []*node.Node
		err    error
	)

	// Without constraints we should get the unfiltered list of nodes back.
	result, err = f.Filter(&cluster.ContainerConfig{}, nodes, true)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)

	// Set a constraint that cannot be fulfilled and expect an error back.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:does_not_exist==true"}}), nodes, true)
	assert.Error(t, err)

	// Set a contraint that can only be filled by a single node.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:name==node1"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// This constraint can only be fulfilled by a subset of nodes.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:group==1"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[2])

	// Validate node pinning by id.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:node==node-2-id"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[2])

	// Validate node pinning by name.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:node==node-1-name"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// Make sure constraints are evaluated as logical ANDs.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:name==node0", "constraint:group==1"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Check matching
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:region==us"}}), nodes, true)
	assert.Error(t, err)
	assert.Len(t, result, 0)

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:region==us*"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:region==*us*"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestConstraintNotExpr(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []*node.Node
		err    error
	)

	// Check not (!) expression
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:name!=node0"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Check not does_not_exist. All should be found
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:name!=does_not_exist"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 4)

	// Check name must not start with n
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:name!=n*"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	// Check not with globber pattern
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:region!=us*"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestConstraintRegExp(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []*node.Node
		err    error
	)

	// Check with regular expression /node\d/ matches node{0..2}
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{`constraint:name==/node\d/`}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Check with regular expression /node\d/ matches node{0..2}
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{`constraint:name==/node[12]/`}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Check with regular expression ! and regexp /node[12]/ matches node[0] and node[3]
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{`constraint:name!=/node[12]/`}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Validate node pinning by ! and regexp.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:node!=/node-[01]-id/"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestFilterRegExpCaseInsensitive(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []*node.Node
		err    error
	)

	// Prepare node with a strange name
	nodes[3].Labels = map[string]string{
		"name":   "aBcDeF",
		"group":  "2",
		"region": "eu",
	}

	// Case-sensitive, so not match
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{`constraint:name==/abcdef/`}}), nodes, true)
	assert.Error(t, err)
	assert.Len(t, result, 0)

	// Match with case-insensitive
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{`constraint:name==/(?i)abcdef/`}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[3])
	assert.Equal(t, result[0].Labels["name"], "aBcDeF")

	// Test ! filter combined with case insensitive
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{`constraint:name!=/(?i)abc*/`}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestFilterEquals(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []*node.Node
		err    error
	)

	// Check == comparison
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:name==node0"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	// Test == with glob
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:region==us*"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Validate node name with ==
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:node==node-1-name"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])
}

func TestUnsupportedOperators(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []*node.Node
		err    error
	)

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:name=node0"}}), nodes, true)
	assert.Error(t, err)
	assert.Len(t, result, 0)

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:name=!node0"}}), nodes, true)
	assert.Error(t, err)
	assert.Len(t, result, 0)
}

func TestFilterSoftConstraint(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []*node.Node
		err    error
	)

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:node==~node-1-name"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{`constraint:name!=~/(?i)abc*/`}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 4)

	// Check not with globber pattern
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:region!=~us*"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:region!=~can*"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 4)

	// Check matching
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:region==~us~"}}), nodes, true)
	assert.Error(t, err)
	assert.Len(t, result, 0)
}
