package filter

import (
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestAffinityFilter(t *testing.T) {
	var (
		f     = AffinityFilter{}
		nodes = []cluster.Node{
			&FakeNode{
				id:   "node-0-id",
				name: "node-0-name",
				addr: "node-0",
				containers: []*cluster.Container{
					{Container: dockerclient.Container{
						Id:    "container-n0-0-id",
						Names: []string{"/container-n0-0-name"},
					}},
					{Container: dockerclient.Container{
						Id:    "container-n0-1-id",
						Names: []string{"/container-n0-1-name"},
					}},
				},
				images: []*cluster.Image{{Image: dockerclient.Image{
					Id:       "image-0-id",
					RepoTags: []string{"image-0:tag1", "image-0:tag2"},
				}}},
			},
			&FakeNode{
				id:   "node-1-id",
				name: "node-1-name",
				addr: "node-1",
				containers: []*cluster.Container{
					{Container: dockerclient.Container{
						Id:    "container-n1-0-id",
						Names: []string{"/container-n1-0-name"},
					}},
					{Container: dockerclient.Container{
						Id:    "container-n1-1-id",
						Names: []string{"/container-n1-1-name"},
					}},
				},
				images: []*cluster.Image{{Image: dockerclient.Image{
					Id:       "image-1-id",
					RepoTags: []string{"image-1:tag1", "image-0:tag3", "image-1:tag2"},
				}}},
			},
			&FakeNode{
				id:   "node-2-id",
				name: "node-2-name",
				addr: "node-2",
			},
		}
		result []cluster.Node
		err    error
	)

	// Without constraints we should get the unfiltered list of nodes back.
	result, err = f.Filter(&dockerclient.ContainerConfig{}, nodes)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)

	// Set a constraint that cannot be fullfilled and expect an error back.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:container==does_not_exsits"},
	}, nodes)
	assert.Error(t, err)

	// Set a contraint that can only be filled by a single node.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:container==container-n0*"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// This constraint can only be fullfilled by a subset of nodes.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:container==container-*"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[2])

	// Validate by id.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:container==container-n0-0-id"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Validate by id.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:container!=container-n0-0-id"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[0])

	// Validate by id.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:container!=container-n0-1-id"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[0])

	// Validate by name.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:container==container-n1-0-name"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// Validate by name.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:container!=container-n1-0-name"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[1])

	// Validate by name.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:container!=container-n1-1-name"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[1])

	// Validate images by id
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:image==image-0-id"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Validate images by name
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:image==image-0:tag3"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// Validate images by name
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:image!=image-0:tag3"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Validate images by name
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:image==image-1"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// Validate images by name
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:image!=image-1"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Ensure that constraints can be chained.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{
			"affinity:container!=container-n0-1-id",
			"affinity:container!=container-n1-1-id",
		},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[2])

	// Ensure that constraints can be chained.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{
			"affinity:container==container-n0-1-id",
			"affinity:container==container-n1-1-id",
		},
	}, nodes)
	assert.Error(t, err)

	//Tests for Soft affinity
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:image==~image-0:tag3"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:image==~image-1:tag3"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:image==~image-*"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:image!=~image-*"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[2])

	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:image==~/image-\\d*/"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Not support = any more
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:image=image-0:tag3"},
	}, nodes)
	assert.Error(t, err)
	assert.Len(t, result, 0)

	// Not support =! any more
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:image=!image-0:tag3"},
	}, nodes)
	assert.Error(t, err)
	assert.Len(t, result, 0)

}
