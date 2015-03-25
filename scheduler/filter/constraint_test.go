package filter

import (
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func testFixtures() []cluster.Node {
	return []cluster.Node{
		&FakeNode{
			id:   "node-0-id",
			name: "node-0-name",
			addr: "node-0",
			labels: map[string]string{
				"name":   "node0",
				"group":  "1",
				"region": "us-west",
			},
		},

		&FakeNode{
			id:   "node-1-id",
			name: "node-1-name",
			addr: "node-1",
			labels: map[string]string{
				"name":   "node1",
				"group":  "1",
				"region": "us-east",
			},
		},

		&FakeNode{
			id:   "node-2-id",
			name: "node-2-name",
			addr: "node-2",
			labels: map[string]string{
				"name":   "node2",
				"group":  "2",
				"region": "eu",
			},
		},

		&FakeNode{
			id:   "node-3-id",
			name: "node-3-name",
			addr: "node-3",
		},
	}
}

func TestConstrainteFilter(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []cluster.Node
		err    error
	)

	// Without constraints we should get the unfiltered list of nodes back.
	result, err = f.Filter(&dockerclient.ContainerConfig{}, nodes)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)

	// Set a constraint that cannot be fullfilled and expect an error back.
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:does_not_exist==true"}}, nodes)
	assert.Error(t, err)

	// Set a contraint that can only be filled by a single node.
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:name==node1"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// This constraint can only be fullfilled by a subset of nodes.
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:group==1"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[2])

	// Validate node pinning by id.
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:node==node-2-id"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[2])

	// Validate node pinning by name.
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:node==node-1-name"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// Make sure constraints are evaluated as logical ANDs.
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:name==node0", "constraint:group==1"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Check matching
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:region==us"}}, nodes)
	assert.Error(t, err)
	assert.Len(t, result, 0)

	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:region==us*"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:region==*us*"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestConstraintNotExpr(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []cluster.Node
		err    error
	)

	// Check not (!) expression
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:name!=node0"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Check not does_not_exist. All should be found
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:name!=does_not_exist"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 4)

	// Check name must not start with n
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:name!=n*"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	// Check not with globber pattern
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:region!=us*"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestConstraintRegExp(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []cluster.Node
		err    error
	)

	// Check with regular expression /node\d/ matches node{0..2}
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{`constraint:name==/node\d/`}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Check with regular expression /node\d/ matches node{0..2}
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{`constraint:name==/node[12]/`}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Check with regular expression ! and regexp /node[12]/ matches node[0] and node[3]
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{`constraint:name!=/node[12]/`}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Validate node pinning by ! and regexp.
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:node!=/node-[01]-id/"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestFilterRegExpCaseInsensitive(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []cluster.Node
		err    error
	)

	// Prepare node with a strange name
	if n, ok := nodes[3].(*FakeNode); ok {
		n.labels = map[string]string{
			"name":   "aBcDeF",
			"group":  "2",
			"region": "eu",
		}
	}

	// Case-sensitive, so not match
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{`constraint:name==/abcdef/`}}, nodes)
	assert.Error(t, err)
	assert.Len(t, result, 0)

	// Match with case-insensitive
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{`constraint:name==/(?i)abcdef/`}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[3])
	assert.Equal(t, result[0].Labels()["name"], "aBcDeF")

	// Test ! filter combined with case insensitive
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{`constraint:name!=/(?i)abc*/`}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestFilterEquals(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []cluster.Node
		err    error
	)

	// Check == comparison
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:name==node0"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	// Test == with glob
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:region==us*"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Validate node name with ==
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:node==node-1-name"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])
}

func TestUnsupportedOperators(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []cluster.Node
		err    error
	)

	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:name=node0"}}, nodes)
	assert.Error(t, err)
	assert.Len(t, result, 0)

	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:name=!node0"}}, nodes)
	assert.Error(t, err)
	assert.Len(t, result, 0)
}

func TestFilterSoftConstraint(t *testing.T) {
	var (
		f      = ConstraintFilter{}
		nodes  = testFixtures()
		result []cluster.Node
		err    error
	)

	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:node==~node-1-name"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{`constraint:name!=~/(?i)abc*/`}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 4)

	// Check not with globber pattern
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:region!=~us*"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:region!=~can*"}}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 4)

	// Check matching
	result, err = f.Filter(&dockerclient.ContainerConfig{Env: []string{"constraint:region==~us~"}}, nodes)
	assert.Error(t, err)
	assert.Len(t, result, 0)
}
