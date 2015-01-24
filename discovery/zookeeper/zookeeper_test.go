package zookeeper

import (
	"testing"

	"github.com/docker/swarm/discovery"
	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	service := &ZkDiscoveryService{}

	assert.Equal(t, service.Initialize("127.0.0.1", 0).Error(), "invalid format \"127.0.0.1\", missing <path>")

	assert.Error(t, service.Initialize("127.0.0.1/path", 0))
	assert.Equal(t, service.path, "/path")

	assert.Error(t, service.Initialize("127.0.0.1,127.0.0.2,127.0.0.3/path", 0))
	assert.Equal(t, service.path, "/path")
}

func TestCreateNodes(t *testing.T) {
	service := &ZkDiscoveryService{}
	assert.Equal(t, service.createNodes(nil), []*discovery.Node{})
	nodes := service.createNodes([]string{"127.0.0.1", "127.0.0.2"})
	assert.Equal(t, nodes[0].String(), "127.0.0.1")
	assert.Equal(t, nodes[1].String(), "127.0.0.2")
}
