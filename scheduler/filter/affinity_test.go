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
		nodes = []*cluster.Node{
			cluster.NewNode("node-0", "2375", 0),
			cluster.NewNode("node-1", "2375", 0),
			cluster.NewNode("node-2", "2375", 0),
		}
		result []*cluster.Node
		err    error
	)

	nodes[0].ID = "node-0-id"
	nodes[0].Name = "node-0-name"
	nodes[0].AddContainer(&cluster.Container{
		Container: dockerclient.Container{
			Id:    "container-0-id",
			Names: []string{"container-0-name"},
		},
	})
	nodes[0].AddImage(&dockerclient.Image{
		Id:       "image-0-id",
		RepoTags: []string{"image-0:tag1", "image-0:tag2"},
	})

	nodes[1].ID = "node-1-id"
	nodes[1].Name = "node-1-name"
	nodes[1].AddContainer(&cluster.Container{
		Container: dockerclient.Container{
			Id:    "container-1-id",
			Names: []string{"container-1-name"},
		},
	})
	nodes[1].AddImage(&dockerclient.Image{
		Id:       "image-1-id",
		RepoTags: []string{"image-1:tag1", "image-0:tag3", "image-1:tag2"},
	})

	nodes[2].ID = "node-2-id"
	nodes[2].Name = "node-2-name"

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
		Env: []string{"affinity:container==container-0*"},
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
		Env: []string{"affinity:container==container-0-id"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[0])

	// Validate by id.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:container!=container-0-id"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.NotContains(t, result, nodes[0])

	// Validate by name.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:container==container-1-name"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result[0], nodes[1])

	// Validate by name.
	result, err = f.Filter(&dockerclient.ContainerConfig{
		Env: []string{"affinity:container!=container-1-name"},
	}, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
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
