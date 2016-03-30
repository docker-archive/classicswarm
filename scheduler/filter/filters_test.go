package filter

import (
	"testing"

	"github.com/docker/engine-api/types"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestApplyFilters(t *testing.T) {
	var (
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
				HealthIndicator: 100,
			},
			{
				ID:   "node-1-id",
				Name: "node-1-name",
				Addr: "node-1",
				Containers: []*cluster.Container{
					{
						Container: dockerclient.Container{
							Id:    "container-n1-0-id",
							Names: []string{"/container-n1-0-name"},
						},
					},
					{
						Container: dockerclient.Container{
							Id:    "container-n1-1-id",
							Names: []string{"/container-n1-1-name"},
						},
					},
				},
				Images: []*cluster.Image{{Image: types.Image{
					ID:       "image-1-id",
					RepoTags: []string{"image-1:tag1", "image-0:tag3", "image-1:tag2"},
				}}},
				HealthIndicator: 0,
			},
		}
		result []*node.Node
		err    error
	)

	//Tests for Soft affinity, it should be considered as last
	config := cluster.BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:image==~image-0:tag3"}})
	result, err = ApplyFilters(filters, config, nodes, true)
	assert.Error(t, err)
	assert.Len(t, result, 0)
	result, err = ApplyFilters(filters, config, nodes, false)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

}
