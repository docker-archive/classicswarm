package cluster

import (
	"testing"

	"github.com/docker/engine-api/types"
	dockerfilters "github.com/docker/engine-api/types/filters"
	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	img := Image{}

	img.ID = "378954456789"
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

	img.ID = "378954456789"
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
		{types.Image{ID: "a"}, engine},
		{types.Image{
			ID:     "b",
			Labels: map[string]string{"com.example.project": "bar"},
		}, engine},
		{types.Image{ID: "c"}, engine},
	}

	filters := dockerfilters.NewArgs()
	filters.Add("label", "com.example.project=bar")
	result := images.Filter(ImageFilterOptions{types.ImageListOptions{All: true, Filters: filters}})
	assert.Equal(t, len(result), 1)
	assert.Equal(t, result[0].ID, "b")
}

func TestImagesFilterWithMatchName(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	images := Images{
		{
			types.Image{
				ID:       "a",
				RepoTags: []string{"example:latest", "example:2"},
			},
			engine,
		},
		{
			types.Image{ID: "b", RepoTags: []string{"example:1"}},
			engine,
		},
	}

	result := images.Filter(ImageFilterOptions{types.ImageListOptions{All: true, MatchName: "example:2"}})
	assert.Equal(t, len(result), 1)
	assert.Equal(t, result[0].ID, "a")
}

func TestImagesFilterWithMatchNameWithTag(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	images := Images{
		{
			types.Image{
				ID:       "a",
				RepoTags: []string{"example:latest", "example:2"},
			},
			engine,
		},
		{
			types.Image{ID: "b", RepoTags: []string{"example:1"}},
			engine,
		},
		{
			types.Image{ID: "c", RepoTags: []string{"foo:latest"}},
			engine,
		},
	}

	result := images.Filter(ImageFilterOptions{types.ImageListOptions{All: true, MatchName: "example"}})
	assert.Equal(t, len(result), 2)
}

func TestParseRepositoryTag(t *testing.T) {

	repo, tag := ParseRepositoryTag("localhost.localdomain:5000/samalba/hipache:latest")
	if tag != "latest" {
		t.Errorf("repo=%s tag=%s", repo, tag)
	}
	repo, tag = ParseRepositoryTag("localhost:5000/foo/bar@sha256:bc8813ea7b3603864987522f02a76101c17ad122e1c46d790efc0fca78ca7bfb")
	if tag != "sha256:bc8813ea7b3603864987522f02a76101c17ad122e1c46d790efc0fca78ca7bfb" {
		t.Logf("repo=%s tag=%s", repo, tag)
	}
	repo, tag = ParseRepositoryTag("localhost:5000/foo/bar")
	if tag != "" {
		t.Logf("repo=%s tag=%s", repo, tag)
	}
	repo, tag = ParseRepositoryTag("localhost:5000/foo/bar:latest")
	t.Logf("repo=%s tag=%s", repo, tag)
	if tag != "latest" {
		t.Logf("repo=%s tag=%s", repo, tag)
	}
}
