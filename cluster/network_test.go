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

	filtered := networks.Filter([]string{"network_name"}, []string{"abababab"})
	assert.Equal(t, len(filtered), 2)
	for _, network := range filtered {
		assert.True(t, network.ID == "aaaaaaaaaa1" || network.ID == "ababababab")
	}
}
