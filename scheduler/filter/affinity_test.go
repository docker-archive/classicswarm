package filter

import (
	"testing"

	"github.com/docker/engine-api/types"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestAffinityFilter(t *testing.T) {
	var (
		f     = AffinityFilter{}
		nodes = []*node.Node{
			{
				ID:   "node-0-id",
				Name: "node-0-name",
				Addr: "node-0",
				Containers: []*cluster.Container{
					{Container: dockerclient.Container{
						Id:    "container-n0-0-id",
						Names: []string{"/container-n0-0-name"},
					}},
					{Container: dockerclient.Container{
						Id:    "container-n0-1-id",
						Names: []string{"/container-n0-1-name"},
					}},
				},
				Images: []*cluster.Image{{Image: types.Image{
					ID:       "image-0-id",
					RepoTags: []string{"image-0:tag1", "image-0:tag2"},
				}}},
			},
			{
				ID:   "node-1-id",
				Name: "node-1-name",
				Addr: "node-1",
				Containers: []*cluster.Container{
					{Container: dockerclient.Container{
						Id:    "container-n1-0-id",
						Names: []string{"/container-n1-0-name"},
					}},
					{Container: dockerclient.Container{
						Id:    "container-n1-1-id",
						Names: []string{"/container-n1-1-name"},
					}},
				},
				Images: []*cluster.Image{{Image: types.Image{
					ID:       "image-1-id",
					RepoTags: []string{"image-1:tag1", "image-0:tag3", "image-1:tag2"},
				}}},
			},
			{
				ID:   "node-2-id",
				Name: "node-2-name",
				Addr: "node-2",
			},
		}
		result []*node.Node
		err    error
	)

	// Without constraints we should get the unfiltered list of nodes back.
	result, err = f.Filter(&cluster.ContainerConfig{}, nodes, true)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)

	// Set a constraint that cannot be fulfilled and expect an error back.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:container==does_not_exsits"}}), nodes, true)
	assert.Error(t, err)

	// Set a constraint that can only be filled by a single node.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:container==container-n0*"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// This constraint can only be fulfilled by a subset of nodes.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:container==container-*"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[2])

	// Validate by id.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:container==container-n0-0-id"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Validate by id.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:container!=container-n0-0-id"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[0])

	// Validate by id.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:container!=container-n0-1-id"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[0])

	// Validate by name.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:container==container-n1-0-name"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// Validate by name.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:container!=container-n1-0-name"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[1])

	// Validate by name.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:container!=container-n1-1-name"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NotContains(t, result, nodes[1])

	// Conflicting Constraint
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:container!=container-n1-1-name", "affinity:container==container-n1-1-name"}}), nodes, true)
	assert.Error(t, err)
	assert.Len(t, result, 0)
	// Validate images by id
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image==image-0-id"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Validate images by name
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image==image-0:tag3"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// Validate images by name
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image!=image-0:tag3"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Validate images by name
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image==image-1"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// Validate images by name
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image!=image-1"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Ensure that constraints can be chained.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:container!=container-n0-1-id", "affinity:container!=container-n1-1-id"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[2])

	// Ensure that constraints can be chained.
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:container==container-n0-1-id", "affinity:container==container-n1-1-id"}}), nodes, true)
	assert.Error(t, err)

	//Tests for Soft affinity
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image==~image-0:tag3"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image==~ima~ge-0:tag3"}}), nodes, true)
	assert.Error(t, err)
	assert.Len(t, result, 0)

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image==~image-1:tag3"}}), nodes, true)
	assert.Error(t, err)
	assert.Len(t, result, 0)
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image==~image-1:tag3"}}), nodes, false)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image==~image-*"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image!=~image-*"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[2])

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image==~/image-\\d*/"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Not support = any more
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image=image-0:tag3"}}), nodes, true)
	assert.Error(t, err)
	assert.Len(t, result, 0)

	// Not support =! any more
	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image=!image-0:tag3"}}), nodes, true)
	assert.Error(t, err)
	assert.Len(t, result, 0)

}

func TestAffinityFilterLabels(t *testing.T) {
	var (
		f     = AffinityFilter{}
		nodes = []*node.Node{
			{
				ID:   "node-0-id",
				Name: "node-0-name",
				Addr: "node-0",
				Containers: []*cluster.Container{
					{Container: dockerclient.Container{
						Id:    "container-n0-id",
						Names: []string{"/container-n0-name"},
					}},
				},
				Images: []*cluster.Image{{Image: types.Image{
					ID:       "image-0-id",
					RepoTags: []string{"image-0:tag0"},
				}}},
			},
			{
				ID:   "node-1-id",
				Name: "node-1-name",
				Addr: "node-1",
				Containers: []*cluster.Container{
					{Container: dockerclient.Container{
						Id:    "container-n1-id",
						Names: []string{"/container-n1-name"},
					}},
				},
				Images: []*cluster.Image{{Image: types.Image{
					ID:       "image-1-id",
					RepoTags: []string{"image-1:tag1"},
				}}},
			},
		}
		result []*node.Node
		err    error
	)

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image==image-1"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image!=image-1"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Labels: map[string]string{"com.docker.swarm.affinities": "[\"image==image-1\"]"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	result, err = f.Filter(cluster.BuildContainerConfig(dockerclient.ContainerConfig{Labels: map[string]string{"com.docker.swarm.affinities": "[\"image!=image-1\"]"}}), nodes, true)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])
}
