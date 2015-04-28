package kv

import (
	"testing"

	"github.com/docker/swarm/discovery"
	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	discoveryService := &Discovery{}
	tls := &discovery.TLS{}

	assert.Equal(t, discoveryService.Initialize("127.0.0.1", 0, tls).Error(), "invalid format \"127.0.0.1\", missing <path>")

	assert.Error(t, discoveryService.Initialize("127.0.0.1/path", 0, tls))
	assert.Equal(t, discoveryService.prefix, "path")

	assert.Error(t, discoveryService.Initialize("127.0.0.1,127.0.0.2,127.0.0.3/path", 0, tls))
	assert.Equal(t, discoveryService.prefix, "path")

}
