package cluster

import (
	"testing"

	dockerfilters "github.com/docker/docker/pkg/parsers/filters"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	img := Image{}

	img.Id = "378954456789"
	img.RepoTags = []string{"name:latest"}
	img.RepoDigests = []string{"name@sha256:a973f1415c489a934bf56dd653079d36b4ec717760215645726439de9705911d"}

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

	assert.True(t, img.Match("name@sha256:a973f1415c489a934bf56dd653079d36b4ec717760215645726439de9705911d", true))
	assert.False(t, img.Match("name@sha256:111111415c489a934bf56dd653079d36b4ec717760215645726439de9705911d", true))
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

func TestImagesFilterWithLabelFilter(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	images := Images{
		{dockerclient.Image{Id: "a"}, engine},
		{dockerclient.Image{
			Id:     "b",
			Labels: map[string]string{"com.example.project": "bar"},
		}, engine},
		{dockerclient.Image{Id: "c"}, engine},
	}

	filters := dockerfilters.Args{"label": {"com.example.project=bar"}}
	result := images.Filter(ImageFilterOptions{All: true, Filters: filters})
	assert.Equal(t, len(result), 1)
	assert.Equal(t, result[0].Id, "b")
}

func TestImagesFilterWithNameFilter(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	images := Images{
		{
			dockerclient.Image{
				Id:       "a",
				RepoTags: []string{"example:latest", "example:2"},
			},
			engine,
		},
		{
			dockerclient.Image{Id: "b", RepoTags: []string{"example:1"}},
			engine,
		},
	}

	result := images.Filter(ImageFilterOptions{
		All:        true,
		NameFilter: "example:2",
	})
	assert.Equal(t, len(result), 1)
	assert.Equal(t, result[0].Id, "a")
}

func TestImagesFilterWithNameFilterWithTag(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	images := Images{
		{
			dockerclient.Image{
				Id:       "a",
				RepoTags: []string{"example:latest", "example:2"},
			},
			engine,
		},
		{
			dockerclient.Image{Id: "b", RepoTags: []string{"example:1"}},
			engine,
		},
		{
			dockerclient.Image{Id: "c", RepoTags: []string{"foo:latest"}},
			engine,
		},
	}

	result := images.Filter(ImageFilterOptions{
		All:        true,
		NameFilter: "example",
	})
	assert.Equal(t, len(result), 2)
}
