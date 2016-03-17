package cluster

import (
	"testing"

	"github.com/docker/engine-api/types"
	"github.com/stretchr/testify/assert"
)

func TestNetworksFilter(t *testing.T) {
	engine := &Engine{ID: "id"}
	networks := Networks{
		{types.NetworkResource{
			ID:   "ababababab",
			Name: "something",
		}, engine},
		{types.NetworkResource{
			ID:   "aaaaaaaaaa1",
			Name: "network_name",
		}, engine},
		{types.NetworkResource{
			ID:   "bbbbbbbbbb",
			Name: "somethingelse",
		}, engine},
		{types.NetworkResource{
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
		{types.NetworkResource{
			ID:   "global",
			Name: "global",
			Containers: map[string]types.EndpointResource{
				"c1": {},
			},
		}, engine1},
		{types.NetworkResource{
			ID:   "global",
			Name: "global",
			Containers: map[string]types.EndpointResource{
				"c2": {},
			},
		}, engine2},
		{types.NetworkResource{
			ID:   "local1",
			Name: "local",
			Containers: map[string]types.EndpointResource{
				"c3": {},
			},
		}, engine1},
		{types.NetworkResource{
			ID:   "local2",
			Name: "local",
			Containers: map[string]types.EndpointResource{
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

func TestRemoveDuplicateEndpoints(t *testing.T) {
	engine1 := &Engine{ID: "id1"}
	network := Network{
		types.NetworkResource{
			ID:   "global",
			Name: "voteappbase_voteapp",
			Containers: map[string]types.EndpointResource{
				"028771f7f6a54c486d441ecfc92aad68e0836a1f0a5a0c227c514f14848e2b54": {
					Name:        "voteappbase_worker_1",
					EndpointID:  "49f621862a0659f462870a6cd15874da44592e399f41da2a3019d81b7427315b",
					MacAddress:  "02:42:0a:00:02:04",
					IPv4Address: "10.0.2.4/24",
					IPv6Address: "",
				},
				"6baa9c82be8fe731556b371312989e22fb9e67121698c94b4890dd554381d97b": {
					Name:        "voteappbase_voting-app_1",
					EndpointID:  "95e831e91a76c87a51b93217645fcaf093c209e2b6691493bfdc0cf2c39698d0",
					MacAddress:  "02:42:0a:00:02:05",
					IPv4Address: "10.0.2.5/24",
					IPv6Address: "",
				},
				"995eccf14e33797b15c2c1ba67f68b7ced0754c42f096e02cc1c488a418f6126": {
					Name:        "db",
					EndpointID:  "e881cf9ba6dcc5495630d4e2ffba9178f7f4456f12938624562e2cf79888e6b4",
					MacAddress:  "02:42:0a:00:02:02",
					IPv4Address: "10.0.2.2/24",
					IPv6Address: "",
				},
				"e4547f30d8dcd93a9c0da8b44e2a8f793eb01260aa478243605d137a90688611": {
					Name:        "voteappbase_result-app_1",
					EndpointID:  "d79ed2653731eccc968bd32da158bf74646511242d7a96f31fd350eda5b658cb",
					MacAddress:  "02:42:0a:00:02:06",
					IPv4Address: "10.0.2.6/24",
					IPv6Address: "",
				},
				"ep-49f621862a0659f462870a6cd15874da44592e399f41da2a3019d81b7427315b": {
					Name:        "voteappbase_worker_1",
					EndpointID:  "49f621862a0659f462870a6cd15874da44592e399f41da2a3019d81b7427315b",
					MacAddress:  "02:42:0a:00:02:04",
					IPv4Address: "10.0.2.4/24",
					IPv6Address: "",
				},
				"ep-95e831e91a76c87a51b93217645fcaf093c209e2b6691493bfdc0cf2c39698d0": {
					Name:        "voteappbase_voting-app_1",
					EndpointID:  "95e831e91a76c87a51b93217645fcaf093c209e2b6691493bfdc0cf2c39698d0",
					MacAddress:  "02:42:0a:00:02:05",
					IPv4Address: "10.0.2.5/24",
					IPv6Address: "",
				},
				"ep-d79ed2653731eccc968bd32da158bf74646511242d7a96f31fd350eda5b658cb": {
					Name:        "voteappbase_result-app_1",
					EndpointID:  "d79ed2653731eccc968bd32da158bf74646511242d7a96f31fd350eda5b658cb",
					MacAddress:  "02:42:0a:00:02:06",
					IPv4Address: "10.0.2.6/24",
					IPv6Address: "",
				},
				"ep-e881cf9ba6dcc5495630d4e2ffba9178f7f4456f12938624562e2cf79888e6b4": {
					Name:        "db",
					EndpointID:  "e881cf9ba6dcc5495630d4e2ffba9178f7f4456f12938624562e2cf79888e6b4",
					MacAddress:  "02:42:0a:00:02:02",
					IPv4Address: "10.0.2.2/24",
					IPv6Address: "",
				},
				"ep-eaa1cf9ba6dcc5495630d4e2ffba917af7f4456f1293a624562e2cf79aaae6b4": {
					Name:        "db-stale",
					EndpointID:  "eaa1cf9ba6dcc5495630d4e2ffba917af7f4456f1293a624562e2cf79aaae6b4",
					MacAddress:  "02:42:0a:00:02:32",
					IPv4Address: "10.0.2.33/24",
					IPv6Address: "",
				},
				"ep-f2e8540123a5b5894da462b8fd06de6394cb1a263392167b87e1b7195ec33055": {
					Name:        "redis",
					EndpointID:  "f2e8540123a5b5894da462b8fd06de6394cb1a263392167b87e1b7195ec33055",
					MacAddress:  "02:42:0a:00:02:03",
					IPv4Address: "10.0.2.3/24",
					IPv6Address: "",
				},
				"f7e42305c1c50b145814d28508312e7374edbafebd8f04115606e58fde96f441": {
					Name:        "redis",
					EndpointID:  "f2e8540123a5b5894da462b8fd06de6394cb1a263392167b87e1b7195ec33055",
					MacAddress:  "02:42:0a:00:02:03",
					IPv4Address: "10.0.2.3/24",
					IPv6Address: "",
				},
			},
		}, engine1}

	cleanNet := network.RemoveDuplicateEndpoints()
	assert.Equal(t, len(cleanNet.Containers), 6)

	// good endpoints are preserved
	resource, ok := cleanNet.Containers["028771f7f6a54c486d441ecfc92aad68e0836a1f0a5a0c227c514f14848e2b54"]
	assert.True(t, ok)
	assert.Equal(t, resource.EndpointID, "49f621862a0659f462870a6cd15874da44592e399f41da2a3019d81b7427315b")

	resource, ok = cleanNet.Containers["f7e42305c1c50b145814d28508312e7374edbafebd8f04115606e58fde96f441"]
	assert.True(t, ok)
	assert.Equal(t, resource.EndpointID, "f2e8540123a5b5894da462b8fd06de6394cb1a263392167b87e1b7195ec33055")

	// duplicate endpoint should be removed
	resource, ok = cleanNet.Containers["ep-f2e8540123a5b5894da462b8fd06de6394cb1a263392167b87e1b7195ec33055"]
	assert.False(t, ok)

	// stale endpoint is preserved
	resource, ok = cleanNet.Containers["ep-eaa1cf9ba6dcc5495630d4e2ffba917af7f4456f1293a624562e2cf79aaae6b4"]
	assert.True(t, ok)
	assert.Equal(t, resource.EndpointID, "eaa1cf9ba6dcc5495630d4e2ffba917af7f4456f1293a624562e2cf79aaae6b4")
}
