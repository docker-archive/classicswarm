package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	img := Image{}

	img.Id = "378954456789"
	img.RepoTags = []string{"name:latest"}

	assert.True(t, img.Match("378954456789", true))
	assert.True(t, img.Match("3789", true))
	assert.True(t, img.Match("378", true))
	assert.False(t, img.Match("37", true))

	assert.True(t, img.Match("name:latest", true))
	assert.True(t, img.Match("name", true))
	assert.False(t, img.Match("nam", true))
	assert.False(t, img.Match("na", true))

	assert.True(t, img.Match("378954456789", false))
	assert.True(t, img.Match("3789", false))
	assert.True(t, img.Match("378", false))
	assert.False(t, img.Match("37", false))

	assert.True(t, img.Match("name:latest", false))
	assert.True(t, img.Match("name", false))
	assert.False(t, img.Match("nam", false))
	assert.False(t, img.Match("na", false))
}

func TestMatchPrivateRepo(t *testing.T) {
	img := Image{}

	img.Id = "378954456789"
	img.RepoTags = []string{"private.registry.com:5000/name:latest"}

	assert.True(t, img.Match("private.registry.com:5000/name:latest", true))
	assert.True(t, img.Match("private.registry.com:5000/name", true))
	assert.False(t, img.Match("private.registry.com:5000/nam", true))
	assert.False(t, img.Match("private.registry.com:5000/na", true))

	assert.True(t, img.Match("private.registry.com:5000/name", false))
	assert.False(t, img.Match("private.registry.com:5000/nam", false))
	assert.False(t, img.Match("private.registry.com:5000/na", false))
}
