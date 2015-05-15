package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	img := Image{}

	img.Id = "378954456789"
	img.RepoTags = []string{"name:latest"}

	assert.True(t, img.Match("378954456789"))
	assert.True(t, img.Match("3789"))
	assert.True(t, img.Match("378"))
	assert.False(t, img.Match("37"))

	assert.True(t, img.Match("name:latest"))
	assert.False(t, img.Match("name"))
	assert.False(t, img.Match("nam"))
	assert.False(t, img.Match("na"))
}
