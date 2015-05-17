package kv

import (
	"testing"

	"github.com/docker/swarm/pkg/store"
	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	d := &Discovery{backend: store.MOCK}
	assert.EqualError(t, d.Initialize("127.0.0.1", 0), "invalid format \"127.0.0.1\", missing <path>")

	d = &Discovery{backend: store.MOCK}
	assert.NoError(t, d.Initialize("127.0.0.1:1234/path", 0))
	s := d.store.(*store.Mock)
	assert.Len(t, s.Endpoints, 1)
	assert.Equal(t, s.Endpoints[0], "127.0.0.1:1234")
	assert.Equal(t, d.prefix, "path")

	d = &Discovery{backend: store.MOCK}
	assert.NoError(t, d.Initialize("127.0.0.1:1234,127.0.0.2:1234,127.0.0.3:1234/path", 0))
	s = d.store.(*store.Mock)
	assert.Len(t, s.Endpoints, 3)
	assert.Equal(t, s.Endpoints[0], "127.0.0.1:1234")
	assert.Equal(t, s.Endpoints[1], "127.0.0.2:1234")
	assert.Equal(t, s.Endpoints[2], "127.0.0.3:1234")
	assert.Equal(t, d.prefix, "path")
}
