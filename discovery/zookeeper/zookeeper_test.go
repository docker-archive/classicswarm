package zookeeper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	service := &Discovery{}

	assert.Equal(t, service.Initialize("127.0.0.1", 0).Error(), "invalid format \"127.0.0.1\", missing <path>")

	assert.Error(t, service.Initialize("127.0.0.1/path", 0))
	assert.Equal(t, service.fullpath(), "/path")

	assert.Error(t, service.Initialize("127.0.0.1,127.0.0.2,127.0.0.3/path", 0))
	assert.Equal(t, service.fullpath(), "/path")

	assert.Error(t, service.Initialize("127.0.0.1,127.0.0.2,127.0.0.3/path/sub1/sub2", 0))
	assert.Equal(t, service.fullpath(), "/path/sub1/sub2")
}
