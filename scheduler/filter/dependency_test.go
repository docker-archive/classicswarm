package filter

import (
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestDependencyFilterSimple(t *testing.T) {
	var (
		f     = DependencyFilter{}
		nodes = []cluster.Node{
			&FakeNode{
				id:         "node-0-id",
				name:       "node-0-name",
				addr:       "node-0",
				containers: []*cluster.Container{{Container: dockerclient.Container{Id: "c0"}}},
			},

			&FakeNode{
				id:         "node-1-id",
				name:       "node-1-name",
				addr:       "node-1",
				containers: []*cluster.Container{{Container: dockerclient.Container{Id: "c1"}}},
			},

			&FakeNode{
				id:         "node-2-id",
				name:       "node-2-name",
				addr:       "node-2",
				containers: []*cluster.Container{{Container: dockerclient.Container{Id: "c2"}}},
			},
		}
		result []cluster.Node
		err    error
		config *dockerclient.ContainerConfig
	)

	// No dependencies - make sure we don't filter anything out.
	config = &dockerclient.ContainerConfig{}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)

	// volumes-from.
	config = &dockerclient.ContainerConfig{HostConfig: dockerclient.HostConfig{
		VolumesFrom: []string{"c0"},
	}}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// link.
	config = &dockerclient.ContainerConfig{HostConfig: dockerclient.HostConfig{
		Links: []string{"c1:foobar"},
	}}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// net.
	config = &dockerclient.ContainerConfig{HostConfig: dockerclient.HostConfig{
		NetworkMode: "container:c2",
	}}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[2])

	// net not prefixed by "container:" should be ignored.
	config = &dockerclient.ContainerConfig{HostConfig: dockerclient.HostConfig{
		NetworkMode: "bridge",
	}}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)
}

func TestDependencyFilterMulti(t *testing.T) {
	var (
		f     = DependencyFilter{}
		nodes = []cluster.Node{
			// nodes[0] has c0 and c1
			&FakeNode{
				id:   "node-0-id",
				name: "node-0-name",
				addr: "node-0",
				containers: []*cluster.Container{
					{Container: dockerclient.Container{Id: "c0"}},
					{Container: dockerclient.Container{Id: "c1"}},
				},
			},

			// nodes[1] has c2
			&FakeNode{
				id:         "node-1-id",
				name:       "node-1-name",
				addr:       "node-1",
				containers: []*cluster.Container{{Container: dockerclient.Container{Id: "c2"}}},
			},

			// nodes[2] has nothing
			&FakeNode{
				id:   "node-2-id",
				name: "node-2-name",
				addr: "node-2",
			},
		}
		result []cluster.Node
		err    error
		config *dockerclient.ContainerConfig
	)

	// Depend on c0 which is on nodes[0]
	config = &dockerclient.ContainerConfig{HostConfig: dockerclient.HostConfig{
		VolumesFrom: []string{"c0"},
	}}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Depend on c1 which is on nodes[0]
	config = &dockerclient.ContainerConfig{HostConfig: dockerclient.HostConfig{
		VolumesFrom: []string{"c1"},
	}}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Depend on c0 AND c1 which are both on nodes[0]
	config = &dockerclient.ContainerConfig{HostConfig: dockerclient.HostConfig{
		VolumesFrom: []string{"c0", "c1"},
	}}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Depend on c0 AND c2 which are on different nodes.
	config = &dockerclient.ContainerConfig{HostConfig: dockerclient.HostConfig{
		VolumesFrom: []string{"c0", "c2"},
	}}
	result, err = f.Filter(config, nodes)
	assert.Error(t, err)
}

func TestDependencyFilterChaining(t *testing.T) {
	var (
		f     = DependencyFilter{}
		nodes = []cluster.Node{
			// nodes[0] has c0 and c1
			&FakeNode{
				id:   "node-0-id",
				name: "node-0-name",
				addr: "node-0",
				containers: []*cluster.Container{
					{Container: dockerclient.Container{Id: "c0"}},
					{Container: dockerclient.Container{Id: "c1"}},
				},
			},

			// nodes[1] has c2
			&FakeNode{
				id:         "node-1-id",
				name:       "node-1-name",
				addr:       "node-1",
				containers: []*cluster.Container{{Container: dockerclient.Container{Id: "c2"}}},
			},

			// nodes[2] has nothing
			&FakeNode{
				id:   "node-2-id",
				name: "node-2-name",
				addr: "node-2",
			},
		}
		result []cluster.Node
		err    error
		config *dockerclient.ContainerConfig
	)

	// Different dependencies on c0 and c1
	config = &dockerclient.ContainerConfig{
		HostConfig: dockerclient.HostConfig{
			VolumesFrom: []string{"c0"},
			Links:       []string{"c1"},
			NetworkMode: "container:c1",
		},
	}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Different dependencies on c0 and c2
	config = &dockerclient.ContainerConfig{
		HostConfig: dockerclient.HostConfig{
			VolumesFrom: []string{"c0"},
			Links:       []string{"c2"},
			NetworkMode: "container:c1",
		},
	}
	result, err = f.Filter(config, nodes)
	assert.Error(t, err)
}
