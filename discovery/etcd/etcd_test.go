package etcd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	discovery := &EtcdDiscoveryService{}
	assert.Error(t, discovery.Initialize("127.0.0.1/path", 0))
	assert.Equal(t, discovery.path, "/path/")

	assert.Error(t, discovery.Initialize("127.0.0.1,127.0.0.2,127.0.0.3/path", 0))
	assert.Equal(t, discovery.path, "/path/")
}
