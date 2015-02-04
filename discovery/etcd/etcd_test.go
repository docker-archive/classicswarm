package etcd

import (
	"testing"

	"github.com/coreos/go-etcd/etcd"
	"github.com/docker/swarm/discovery"
	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	discovery := &EtcdDiscoveryService{}

	assert.Equal(t, discovery.Initialize("127.0.0.1", 0).Error(), "invalid format \"127.0.0.1\", missing <path>")

	assert.Error(t, discovery.Initialize("127.0.0.1/path", 0))
	assert.Equal(t, discovery.path, "/path/")

	assert.Error(t, discovery.Initialize("127.0.0.1,127.0.0.2,127.0.0.3/path", 0))
	assert.Equal(t, discovery.path, "/path/")
}

func TestCreateEntries(t *testing.T) {
	service := &EtcdDiscoveryService{}

	entries, err := service.createEntries(nil)
	assert.Equal(t, entries, []*discovery.Entry{})
	assert.NoError(t, err)

	entries, err = service.createEntries(etcd.Nodes{&etcd.Node{Value: "127.0.0.1:2375"}, &etcd.Node{Value: "127.0.0.2:2375"}})
	assert.Equal(t, entries[0].String(), "127.0.0.1:2375")
	assert.Equal(t, entries[1].String(), "127.0.0.2:2375")
	assert.NoError(t, err)

	_, err = service.createEntries(etcd.Nodes{&etcd.Node{Value: "127.0.0.1"}, &etcd.Node{Value: "127.0.0.2"}})
	assert.Error(t, err)
}
