package api

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/swarm/cluster"
	"github.com/stretchr/testify/assert"
	"testing"
)

var engines = []*cluster.Engine{
	{ID: "59C8F4EC-3A71-44C7-BA50-0496B4D8E548",
		Name: "swarm-node-0"},
	{ID: "607B1F15-CECA-41B2-9D26-D2626EB14D8D",
		Name: "swarm-node-1"},
}

var mountOne = []types.MountPoint{
	{Type: "volume",
		Name:   "localVolumeOne",
		Driver: "local"},
	{Type: "bind",
		Name:   "bindNFS",
		Driver: ""},
}

var mountTwo = []types.MountPoint{
	{Type: "volume",
		Name:   "sshvolume/withslash",
		Driver: "vieux/sshfs:latest"},
}

var containers = cluster.Containers([]*cluster.Container{{
	Container: types.Container{
		ID:     "container1-id",
		Names:  []string{"container1-name1"},
		Mounts: mountOne,
	},
	Engine: engines[0],
}, {
	Container: types.Container{
		ID:     "container2-id",
		Names:  []string{"container2-name2"},
		Mounts: mountTwo,
	},
	Engine: engines[1],
}})

var volumes = []*types.Volume{
	{
		Name:   "swarm-node-0/localVolumeOne",
		Driver: "local",
	},
	{
		Name:   "swarm-node-1/sshvolume/withslash",
		Driver: "vieux/sshfs:latest",
	},
	{
		Name:   "swarm-node-1/sshvolume_unused/withslash",
		Driver: "vieux/sshfs:latest",
	},
	{
		Name:   "swarm-node-0/localVolume_unused",
		Driver: "local",
	},
	{
		Name:   "no_slash_unused",
		Driver: "vieux/sshfs:latest",
	},
}

func TestGetUsedVolumes(t *testing.T) {
	usedVolumes := getUsedVolumes(containers)
	// Valid checks
	assert.Contains(t, usedVolumes, "swarm-node-0"+"/"+"localVolumeOne")
	assert.Contains(t, usedVolumes, "swarm-node-1"+"/"+"sshvolume/withslash")
	// Invalid checks
	assert.NotContains(t, usedVolumes, "swarm-node-0"+"/"+"bindNFS")
}

func TestGetFilteredVolumes(t *testing.T) {
	//Get dangling volumes
	danglingVolumes := getFilteredVolumes(containers, volumes, true)

	// Valid checks
	assert.Contains(t, danglingVolumes, volumes[2])
	assert.Contains(t, danglingVolumes, volumes[3])
	assert.Contains(t, danglingVolumes, volumes[4])
	// Invalid checks
	assert.NotContains(t, danglingVolumes, volumes[0])
	assert.NotContains(t, danglingVolumes, volumes[1])

	//Get used volumes
	usedVolumes := getFilteredVolumes(containers, volumes, false)
	// Valid checks
	assert.Contains(t, usedVolumes, volumes[0])
	assert.Contains(t, usedVolumes, volumes[1])
	// Invalid checks
	assert.NotContains(t, usedVolumes, volumes[2])
	assert.NotContains(t, usedVolumes, volumes[3])
}
