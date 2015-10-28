package cluster

import (
	"strings"

	"github.com/docker/docker/pkg/parsers"
	dockerfilters "github.com/docker/docker/pkg/parsers/filters"
	"github.com/samalba/dockerclient"
)

// Image is exported
type Image struct {
	dockerclient.Image

	Engine *Engine
}

// Match is exported
func (image *Image) Match(IDOrName string, matchTag bool) bool {
	size := len(IDOrName)

	// TODO: prefix match can cause false positives with image names
	if image.Id == IDOrName || (size > 2 && strings.HasPrefix(image.Id, IDOrName)) {
		return true
	}

	repoName, tag := parsers.ParseRepositoryTag(IDOrName)

	for _, imageRepoTag := range image.RepoTags {
		imageRepoName, imageTag := parsers.ParseRepositoryTag(imageRepoTag)

		if matchTag == false && imageRepoName == repoName {
			return true
		}
		if imageRepoName == repoName && (imageTag == tag || tag == "") {
			return true
		}
	}
	return false
}

// ImageFilterOptions are the set of filtering options supported by
// Images.Filter()
type ImageFilterOptions struct {
	All        bool
	NameFilter string
	Filters    dockerfilters.Args
}

// Images is a collection of Image objects that can be filtered
type Images []*Image

// Filter returns a new sequence of Images filtered to only the images that
// matched the filtering paramters
func (images Images) Filter(opts ImageFilterOptions) Images {
	includeAll := func(image *Image) bool {
		// TODO: this is wrong if RepoTags == []
		return opts.All || (len(image.RepoTags) != 0 && image.RepoTags[0] != "<none>:<none>")
	}

	includeFilter := func(image *Image) bool {
		if opts.Filters == nil {
			return true
		}
		return opts.Filters.MatchKVList("label", image.Labels)
	}

	includeRepoFilter := func(image *Image) bool {
		if opts.NameFilter == "" {
			return true
		}
		for _, repoTag := range image.RepoTags {
			repoName, _ := parsers.ParseRepositoryTag(repoTag)
			if repoTag == opts.NameFilter || repoName == opts.NameFilter {
				return true
			}
		}
		return false
	}

	filtered := make([]*Image, 0, len(images))
	for _, image := range images {
		if includeAll(image) && includeFilter(image) && includeRepoFilter(image) {
			filtered = append(filtered, image)
		}
	}
	return filtered
}
