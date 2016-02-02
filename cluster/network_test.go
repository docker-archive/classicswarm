package cluster

import (
	"testing"

	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestNetworksFilter(t *testing.T) {
	engine := &Engine{ID: "id"}
	networks := Networks{
		{dockerclient.NetworkResource{
			ID:   "ababababab",
			Name: "something",
		}, engine},
		{dockerclient.NetworkResource{
			ID:   "aaaaaaaaaa1",
			Name: "network_name",
		}, engine},
		{dockerclient.NetworkResource{
			ID:   "bbbbbbbbbb",
			Name: "somethingelse",
		}, engine},
		{dockerclient.NetworkResource{
			ID:   "aaaaaaaaa2",
			Name: "foo",
		}, engine},
	}

	filtered := networks.Filter([]string{"network_name"}, []string{"abababab"}, nil)
	assert.Equal(t, len(filtered), 2)
	for _, network := range filtered {
		assert.True(t, network.ID == "aaaaaaaaaa1" || network.ID == "ababababab")
	}
}

func TestNetworkUniq(t *testing.T) {
	engine1 := &Engine{ID: "id1"}
	engine2 := &Engine{ID: "id2"}
	networks := Networks{
		{dockerclient.NetworkResource{
			ID:   "global",
			Name: "global",
			Containers: map[string]dockerclient.EndpointResource{
				"c1": {},
			},
		}, engine1},
		{dockerclient.NetworkResource{
			ID:   "global",
			Name: "global",
			Containers: map[string]dockerclient.EndpointResource{
				"c2": {},
			},
		}, engine2},
		{dockerclient.NetworkResource{
			ID:   "local1",
			Name: "local",
			Containers: map[string]dockerclient.EndpointResource{
				"c3": {},
			},
		}, engine1},
		{dockerclient.NetworkResource{
			ID:   "local2",
			Name: "local",
			Containers: map[string]dockerclient.EndpointResource{
				"c4": {},
			},
		}, engine2},
	}

	global := networks.Uniq().Get("global")
	assert.NotNil(t, global)
	assert.Equal(t, 2, len(global.Containers))

	local1 := networks.Uniq().Get("local1")
	assert.NotNil(t, local1)
	assert.Equal(t, 1, len(local1.Containers))

	local3 := networks.Uniq().Get("local3")
	assert.Nil(t, local3)
}
