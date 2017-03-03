package filter

import (
	"testing"

	containertypes "github.com/docker/docker/api/types/container"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
	"github.com/stretchr/testify/assert"
)

func TestWhitelistFilter(t *testing.T) {
	var (
		f      = WhitelistFilter{}
		nodes  = testFixtures() // Reuse the same fixtures as the constraint test
		result []*node.Node
		err    error
	)

	// Without a whitelist we should get all nodes back
	result, err = f.Filter(&cluster.ContainerConfig{}, nodes, true)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)

	// Set a multi-node whitelist that cannot be fulfilled and expect an error back.
	result, err = f.Filter(cluster.BuildContainerConfig(containertypes.Config{Env: []string{"whitelist:node==node-5-name|node-6-name|node-7-name"}}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}), nodes, true)
	assert.Error(t, err)

	// Set a single-node whitelist that cannot be fulfilled and expect an error back.
	result, err = f.Filter(cluster.BuildContainerConfig(containertypes.Config{Env: []string{"whitelist:node==node-5-name"}}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}), nodes, true)
	assert.Error(t, err)

	// Set a single-node whitelist that can be fulfilled
	result, err = f.Filter(cluster.BuildContainerConfig(containertypes.Config{Env: []string{"whitelist:node==node-1-name"}}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// Set a multi-node whitelist where all nodes can be fulfilled
	result, err = f.Filter(cluster.BuildContainerConfig(containertypes.Config{Env: []string{"whitelist:node==node-1-name|node-2-name"}}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[0])
	assert.NotContains(t, result, nodes[3])

	// Set a multi-node whitelist where only some of the nodes can be fulfilled
	result, err = f.Filter(cluster.BuildContainerConfig(containertypes.Config{Env: []string{"whitelist:node==node-1-name|node-2-name|node-5-name|node-6-name"}}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[0])
	assert.NotContains(t, result, nodes[3])

	// Make sure we only the intersection of multiple whitelists can be fulfilled
	result, err = f.Filter(cluster.BuildContainerConfig(containertypes.Config{Env: []string{"whitelist:node==node-0-name|node-1-name|node-2-name", "whitelist:node==node-1-name|node-2-name|node-3-name"}}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[0])
	assert.NotContains(t, result, nodes[3])
}
