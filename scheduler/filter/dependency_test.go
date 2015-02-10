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
		nodes = []*cluster.Node{
			cluster.NewNode("node-1", 0),
			cluster.NewNode("node-2", 0),
			cluster.NewNode("node-3", 0),
		}
		result    []*cluster.Node
		err       error
		container *cluster.Container
		config    *dockerclient.ContainerConfig
	)

	container = &cluster.Container{Container: dockerclient.Container{Id: "c0"}}
	assert.NoError(t, nodes[0].AddContainer(container))

	container = &cluster.Container{Container: dockerclient.Container{Id: "c1"}}
	assert.NoError(t, nodes[1].AddContainer(container))

	container = &cluster.Container{Container: dockerclient.Container{Id: "c2"}}
	assert.NoError(t, nodes[2].AddContainer(container))

	// No dependencies - make sure we don't filter anything out.
	config = &dockerclient.ContainerConfig{}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)

	// volumes-from.
	config = &dockerclient.ContainerConfig{
		HostConfig: dockerclient.HostConfig{
			VolumesFrom: []string{"c0"},
		},
	}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// link.
	config = &dockerclient.ContainerConfig{
		HostConfig: dockerclient.HostConfig{
			Links: []string{"c1:foobar"},
		},
	}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// net.
	config = &dockerclient.ContainerConfig{
		HostConfig: dockerclient.HostConfig{
			NetworkMode: "container:c2",
		},
	}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[2])

	// net not prefixed by "container:" should be ignored.
	config = &dockerclient.ContainerConfig{
		HostConfig: dockerclient.HostConfig{
			NetworkMode: "bridge",
		},
	}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)
}

func TestDependencyFilterMulti(t *testing.T) {
	var (
		f     = DependencyFilter{}
		nodes = []*cluster.Node{
			cluster.NewNode("node-1", 0),
			cluster.NewNode("node-2", 0),
			cluster.NewNode("node-3", 0),
		}
		result    []*cluster.Node
		err       error
		container *cluster.Container
		config    *dockerclient.ContainerConfig
	)

	// nodes[0] has c0 and c1
	container = &cluster.Container{Container: dockerclient.Container{Id: "c0"}}
	assert.NoError(t, nodes[0].AddContainer(container))
	container = &cluster.Container{Container: dockerclient.Container{Id: "c1"}}
	assert.NoError(t, nodes[0].AddContainer(container))

	// nodes[1] has c2
	container = &cluster.Container{Container: dockerclient.Container{Id: "c2"}}
	assert.NoError(t, nodes[1].AddContainer(container))

	// nodes[2] has nothing

	// Depend on c0 which is on nodes[0]
	config = &dockerclient.ContainerConfig{
		HostConfig: dockerclient.HostConfig{
			VolumesFrom: []string{"c0"},
		},
	}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Depend on c1 which is on nodes[0]
	config = &dockerclient.ContainerConfig{
		HostConfig: dockerclient.HostConfig{
			VolumesFrom: []string{"c1"},
		},
	}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Depend on c0 AND c1 which are both on nodes[0]
	config = &dockerclient.ContainerConfig{
		HostConfig: dockerclient.HostConfig{
			VolumesFrom: []string{"c0", "c1"},
		},
	}
	result, err = f.Filter(config, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Depend on c0 AND c2 which are on different nodes.
	config = &dockerclient.ContainerConfig{
		HostConfig: dockerclient.HostConfig{
			VolumesFrom: []string{"c0", "c2"},
		},
	}
	result, err = f.Filter(config, nodes)
	assert.Error(t, err)
}

func TestDependencyFilterChaining(t *testing.T) {
	var (
		f     = DependencyFilter{}
		nodes = []*cluster.Node{
			cluster.NewNode("node-1", 0),
			cluster.NewNode("node-2", 0),
			cluster.NewNode("node-3", 0),
		}
		result    []*cluster.Node
		err       error
		container *cluster.Container
		config    *dockerclient.ContainerConfig
	)

	// nodes[0] has c0 and c1
	container = &cluster.Container{Container: dockerclient.Container{Id: "c0"}}
	assert.NoError(t, nodes[0].AddContainer(container))
	container = &cluster.Container{Container: dockerclient.Container{Id: "c1"}}
	assert.NoError(t, nodes[0].AddContainer(container))

	// nodes[1] has c2
	container = &cluster.Container{Container: dockerclient.Container{Id: "c2"}}
	assert.NoError(t, nodes[1].AddContainer(container))

	// nodes[2] has nothing

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
