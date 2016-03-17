package cluster

import (
	"testing"

	"github.com/docker/engine-api/types"
	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	engines := []*Engine{
		{ID: "59C8F4EC-3A71-44C7-BA50-0496B4D8E548",
			Name: "swarm-node-0"},
		{ID: "607B1F15-CECA-41B2-9D26-D2626EB14D8D",
			Name: "swarm-node-1"},
	}
	volumes := Volumes{
		&Volume{
			Volume: types.Volume{
				Name:   "t1",
				Driver: "local",
			},
			Engine: engines[0],
		},
		&Volume{
			Volume: types.Volume{
				Name:   "t2",
				Driver: "local",
			},
			Engine: engines[0],
		},
		&Volume{
			Volume: types.Volume{
				Name:   "t3",
				Driver: "rexray",
			},
			Engine: engines[0],
		},
		&Volume{
			Volume: types.Volume{
				Name:   "t4",
				Driver: "flocker",
			},
			Engine: engines[0],
		},
		&Volume{
			Volume: types.Volume{
				Name:   "t1",
				Driver: "local",
			},
			Engine: engines[1],
		},
		&Volume{
			Volume: types.Volume{
				Name:   "t2",
				Driver: "local",
			},
			Engine: engines[1],
		},
		&Volume{
			Volume: types.Volume{
				Name:   "t3",
				Driver: "rexray",
			},
			Engine: engines[1],
		},
		&Volume{
			Volume: types.Volume{
				Name:   "t4",
				Driver: "flocker",
			},
			Engine: engines[1],
		},
	}
	emptyVolumes := Volumes{}
	assert.Nil(t, emptyVolumes.Get("test"))
	assert.Nil(t, volumes.Get("t1"))
	if assert.NotNil(t, volumes.Get(engines[0].ID+"/"+"t1")) {
		assert.Equal(t, volumes.Get(engines[0].ID+"/"+"t1"), volumes[0])
	}
	if assert.NotNil(t, volumes.Get(engines[0].Name+"/"+"t1")) {
		assert.Equal(t, volumes.Get(engines[0].Name+"/"+"t1"), volumes[0])
	}
	if assert.NotNil(t, volumes.Get("t3")) {
		assert.Equal(t, volumes.Get("t3"), volumes[2])
	}
	if assert.NotNil(t, volumes.Get("t4")) {
		assert.Equal(t, volumes.Get("t4"), volumes[3])
	}
}
