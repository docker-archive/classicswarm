package api

import (
	"testing"

	"github.com/docker/docker/api/types/network"
	"github.com/stretchr/testify/assert"
)

func TestStripNodeNamesFromNetworkingConfig(t *testing.T) {
	networkingConfig := network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"testnet1":               {},
			"nodename/testnet2":      {},
			"nodename/test/net3":     {},
			"foo/test/net4":          {},
			"nodename/nodename/net5": {},
		},
	}
	networkingConfig = stripNodeNamesFromNetworkingConfig(networkingConfig, []string{"nodename"})
	assert.Contains(t, networkingConfig.EndpointsConfig, "testnet1")
	assert.Contains(t, networkingConfig.EndpointsConfig, "testnet2")
	assert.Contains(t, networkingConfig.EndpointsConfig, "test/net3")
	assert.Contains(t, networkingConfig.EndpointsConfig, "foo/test/net4")
	assert.Contains(t, networkingConfig.EndpointsConfig, "nodename/net5")
}
