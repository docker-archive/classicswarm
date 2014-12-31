package consul

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	discovery := &ConsulDiscoveryService{}
	discovery.Initialize("127.0.0.1:8500/path", 0)
	assert.Equal(t, discovery.prefix, "path/")
}
