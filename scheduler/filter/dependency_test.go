package filter

import (
	"testing"

	"github.com/docker/engine-api/types"
	containertypes "github.com/docker/engine-api/types/container"
	networktypes "github.com/docker/engine-api/types/network"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
	"github.com/stretchr/testify/assert"
)

func TestDependencyFilterSimple(t *testing.T) {
	var (
		f     = DependencyFilter{}
		nodes = []*node.Node{
			{
				ID:   "node-0-id",
				Name: "node-0-name",
				Addr: "node-0",
				Containers: []*cluster.Container{{
					Container: types.Container{ID: "c0"},
					Config:    &cluster.ContainerConfig{},
				}},
			},

			{
				ID:   "node-1-id",
				Name: "node-1-name",
				Addr: "node-1",
				Containers: []*cluster.Container{{
					Container: types.Container{ID: "c1"},
					Config:    &cluster.ContainerConfig{},
				}},
			},

			{
				ID:   "node-2-id",
				Name: "node-2-name",
				Addr: "node-2",
				Containers: []*cluster.Container{{
					Container: types.Container{ID: "c2"},
					Config:    &cluster.ContainerConfig{},
				}},
			},
		}
		result []*node.Node
		err    error
		config *cluster.ContainerConfig
	)

	// No dependencies - make sure we don't filter anything out.
	config = &cluster.ContainerConfig{}
	result, err = f.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)

	// volumes-from.
	config = &cluster.ContainerConfig{
		Config: containertypes.Config{},
		HostConfig: containertypes.HostConfig{
			VolumesFrom: []string{"c0"},
		},
		NetworkingConfig: networktypes.NetworkingConfig{}}
	result, err = f.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// volumes-from:rw
	config = &cluster.ContainerConfig{
		Config: containertypes.Config{},
		HostConfig: containertypes.HostConfig{
			VolumesFrom: []string{"c0:rw"},
		},
		NetworkingConfig: networktypes.NetworkingConfig{}}
	result, err = f.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// volumes-from:ro
	config = &cluster.ContainerConfig{
		Config: containertypes.Config{},
		HostConfig: containertypes.HostConfig{
			VolumesFrom: []string{"c0:ro"},
		},
		NetworkingConfig: networktypes.NetworkingConfig{}}
	result, err = f.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// link.
	config = &cluster.ContainerConfig{
		Config: containertypes.Config{},
		HostConfig: containertypes.HostConfig{
			Links: []string{"c1:foobar"},
		},
		NetworkingConfig: networktypes.NetworkingConfig{}}
	result, err = f.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// net.
	config = &cluster.ContainerConfig{
		Config: containertypes.Config{},
		HostConfig: containertypes.HostConfig{
			NetworkMode: containertypes.NetworkMode("container:c2"),
		},
		NetworkingConfig: networktypes.NetworkingConfig{}}
	result, err = f.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[2])

	// net not prefixed by "container:" should be ignored.
	config = &cluster.ContainerConfig{
		Config: containertypes.Config{},
		HostConfig: containertypes.HostConfig{
			NetworkMode: containertypes.NetworkMode("bridge"),
		},
		NetworkingConfig: networktypes.NetworkingConfig{}}
	result, err = f.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)
}

func TestDependencyFilterMulti(t *testing.T) {
	var (
		f     = DependencyFilter{}
		nodes = []*node.Node{
			// nodes[0] has c0 and c1
			{
				ID:   "node-0-id",
				Name: "node-0-name",
				Addr: "node-0",
				Containers: []*cluster.Container{
					{
						Container: types.Container{ID: "c0"},
						Config:    &cluster.ContainerConfig{},
					},
					{
						Container: types.Container{ID: "c1"},
						Config:    &cluster.ContainerConfig{},
					},
				},
			},

			// nodes[1] has c2
			{
				ID:   "node-1-id",
				Name: "node-1-name",
				Addr: "node-1",
				Containers: []*cluster.Container{
					{
						Container: types.Container{ID: "c2"},
						Config:    &cluster.ContainerConfig{},
					},
				},
			},

			// nodes[2] has nothing
			{
				ID:   "node-2-id",
				Name: "node-2-name",
				Addr: "node-2",
			},
		}
		result []*node.Node
		err    error
		config *cluster.ContainerConfig
	)

	// Depend on c0 which is on nodes[0]
	config = &cluster.ContainerConfig{
		Config: containertypes.Config{},
		HostConfig: containertypes.HostConfig{
			VolumesFrom: []string{"c0"},
		},
		NetworkingConfig: networktypes.NetworkingConfig{}}
	result, err = f.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Depend on c1 which is on nodes[0]
	config = &cluster.ContainerConfig{
		Config: containertypes.Config{},
		HostConfig: containertypes.HostConfig{
			VolumesFrom: []string{"c1"},
		},
		NetworkingConfig: networktypes.NetworkingConfig{}}
	result, err = f.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Depend on c0 AND c1 which are both on nodes[0]
	config = &cluster.ContainerConfig{
		Config: containertypes.Config{},
		HostConfig: containertypes.HostConfig{
			VolumesFrom: []string{"c0", "c1"},
		},
		NetworkingConfig: networktypes.NetworkingConfig{}}
	result, err = f.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Depend on c0 AND c2 which are on different nodes.
	config = &cluster.ContainerConfig{
		Config: containertypes.Config{},
		HostConfig: containertypes.HostConfig{
			VolumesFrom: []string{"c0", "c2"},
		},
		NetworkingConfig: networktypes.NetworkingConfig{}}
	result, err = f.Filter(config, nodes, true)
	assert.Error(t, err)
}

func TestDependencyFilterChaining(t *testing.T) {
	var (
		f     = DependencyFilter{}
		nodes = []*node.Node{
			// nodes[0] has c0 and c1
			{
				ID:   "node-0-id",
				Name: "node-0-name",
				Addr: "node-0",
				Containers: []*cluster.Container{
					{
						Container: types.Container{ID: "c0"},
						Config:    &cluster.ContainerConfig{},
					},
					{
						Container: types.Container{ID: "c1"},
						Config:    &cluster.ContainerConfig{},
					},
				},
			},

			// nodes[1] has c2
			{
				ID:   "node-1-id",
				Name: "node-1-name",
				Addr: "node-1",
				Containers: []*cluster.Container{
					{
						Container: types.Container{ID: "c2"},
						Config:    &cluster.ContainerConfig{},
					},
				},
			},

			// nodes[2] has nothing
			{
				ID:   "node-2-id",
				Name: "node-2-name",
				Addr: "node-2",
			},
		}
		result []*node.Node
		err    error
		config *cluster.ContainerConfig
	)

	// Different dependencies on c0 and c1
	config = &cluster.ContainerConfig{
		Config: containertypes.Config{},
		HostConfig: containertypes.HostConfig{
			VolumesFrom: []string{"c0"},
			Links:       []string{"c1"},
			NetworkMode: containertypes.NetworkMode("container:c1"),
		},
		NetworkingConfig: networktypes.NetworkingConfig{}}
	result, err = f.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Different dependencies on c0 and c2
	config = &cluster.ContainerConfig{
		Config: containertypes.Config{},
		HostConfig: containertypes.HostConfig{
			VolumesFrom: []string{"c0"},
			Links:       []string{"c2"},
			NetworkMode: containertypes.NetworkMode("container:c1"),
		},
		NetworkingConfig: networktypes.NetworkingConfig{}}
	result, err = f.Filter(config, nodes, true)
	assert.Error(t, err)
}
