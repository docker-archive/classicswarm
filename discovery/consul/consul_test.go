package consul

import (
	"testing"

	consul "github.com/armon/consul-api"
	"github.com/docker/swarm/discovery"
	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	discovery := &ConsulDiscoveryService{}

	assert.Equal(t, discovery.Initialize("127.0.0.1", 0).Error(), "invalid format \"127.0.0.1\", missing <path>")

	assert.Error(t, discovery.Initialize("127.0.0.1/path", 0))
	assert.Equal(t, discovery.prefix, "path/")

	assert.Error(t, discovery.Initialize("127.0.0.1,127.0.0.2,127.0.0.3/path", 0))
	assert.Equal(t, discovery.prefix, "path/")

}

func TestCreateEntries(t *testing.T) {
	service := &ConsulDiscoveryService{prefix: "prefix"}

	entries, err := service.createEntries(nil)
	assert.Equal(t, entries, []*discovery.Entry{})
	assert.NoError(t, err)

	entries, err = service.createEntries(consul.KVPairs{&consul.KVPair{Value: []byte("127.0.0.1:2375")}, &consul.KVPair{Value: []byte("127.0.0.2:2375")}})
	assert.Equal(t, len(entries), 2)
	assert.Equal(t, entries[0].String(), "127.0.0.1:2375")
	assert.Equal(t, entries[1].String(), "127.0.0.2:2375")
	assert.NoError(t, err)

	_, err = service.createEntries(consul.KVPairs{&consul.KVPair{Value: []byte("127.0.0.1")}, &consul.KVPair{Value: []byte("127.0.0.2")}})
	assert.Error(t, err)
}
