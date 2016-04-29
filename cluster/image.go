package cluster

import (
	"strings"

	"github.com/docker/engine-api/types"
)

// Image is exported
type Image struct {
	types.Image

	Engine *Engine
}

// ParseRepositoryTag gets a repos name and returns the right reposName + tag|digest
// The tag can be confusing because of a port in a repository name.
//     Ex: localhost.localdomain:5000/samalba/hipache:latest
//     Digest ex: localhost:5000/foo/bar@sha256:bc8813ea7b3603864987522f02a76101c17ad122e1c46d790efc0fca78ca7bfb
func ParseRepositoryTag(repos string) (string, string) {
	n := strings.Index(repos, "@")
	if n >= 0 {
		parts := strings.Split(repos, "@")
		return parts[0], parts[1]
	}
	n = strings.LastIndex(repos, ":")
	if n < 0 {
		return repos, ""
	}
	if tag := repos[n+1:]; !strings.Contains(tag, "/") {
		return repos[:n], tag
	}
	return repos, ""
}

// Match is exported
func (image *Image) Match(IDOrName string, matchTag bool) bool {
	size := len(IDOrName)

	// TODO: prefix match can cause false positives with image names
	if image.ID == IDOrName || (size > 2 && strings.HasPrefix(image.ID, IDOrName)) {
		return true
	}

	// trim sha256: and retry
	if parts := strings.SplitN(image.ID, ":", 2); len(parts) == 2 {
		if parts[1] == IDOrName || (size > 2 && strings.HasPrefix(parts[1], IDOrName)) {
			return true
		}
	}

	repoName, tag := ParseRepositoryTag(IDOrName)

	// match repotag
	for _, imageRepoTag := range image.RepoTags {
		imageRepoName, imageTag := ParseRepositoryTag(imageRepoTag)

		if matchTag == false && imageRepoName == repoName {
			return true
		}
		if imageRepoName == repoName && (imageTag == tag || tag == "") {
			return true
		}
	}

	// match repodigests
	for _, imageDigest := range image.RepoDigests {
		imageRepoName, imageDigest := ParseRepositoryTag(imageDigest)

		if matchTag == false && imageRepoName == repoName {
			return true
		}
		if imageRepoName == repoName && (imageDigest == tag || tag == "") {
			return true
		}
	}
	return false
}

// ImageFilterOptions is the set of filtering options supported by
// Images.Filter()
type ImageFilterOptions struct {
	types.ImageListOptions
}

// Images is a collection of Image objects that can be filtered
type Images []*Image

// Filter returns a new sequence of Images filtered to only the images that
// matched the filtering parameters
func (images Images) Filter(opts ImageFilterOptions) Images {
	includeAll := func(image *Image) bool {
		// TODO: this is wrong if RepoTags == []
		return opts.All ||
			(len(image.RepoTags) != 0 && image.RepoTags[0] != "<none>:<none>") ||
			(len(image.RepoDigests) != 0 && image.RepoDigests[0] != "<none>@<none>")
	}

	includeFilter := func(image *Image) bool {
		if opts.Filters.Len() == 0 {
			return true
		}
		return opts.Filters.MatchKVList("label", image.Labels)
	}

	includeRepoFilter := func(image *Image) bool {
		if opts.MatchName == "" {
			return true
		}
		for _, repoTag := range image.RepoTags {
			repoName, _ := ParseRepositoryTag(repoTag)
			if repoTag == opts.MatchName || repoName == opts.MatchName {
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
