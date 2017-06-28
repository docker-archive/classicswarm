package cluster

import (
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
)

// Image is exported
type Image struct {
	types.ImageSummary

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
			(len(image.RepoDigests) != 0 && image.RepoDigests[0] != "<none>@<none>") ||
			opts.Filters.Include("dangling") // delegate dangling filter to decide
	}

	includeFilter := func(image *Image) bool {
		if opts.Filters.Len() == 0 {
			return true
		}
		return opts.Filters.MatchKVList("label", image.Labels)
	}

	referenceFilter := func(image *Image, filter string) bool {
		if !opts.Filters.Include(filter) {
			return true
		}
		candidates := map[string]struct{}{}
		for _, repoTag := range image.RepoTags {
			imageName, _ := ParseRepositoryTag(repoTag)
			candidates[repoTag] = struct{}{}
			candidates[imageName] = struct{}{}
		}
		for _, repoDigests := range image.RepoDigests {
			imageName, _ := ParseRepositoryTag(repoDigests)
			candidates[repoDigests] = struct{}{}
			candidates[imageName] = struct{}{}
		}
		for candidate := range candidates {
			for _, pattern := range opts.Filters.Get(filter) {
				ref, err := reference.Parse(candidate)
				if err != nil {
					continue
				}
				found, matchErr := reference.FamiliarMatch(pattern, ref)
				if matchErr == nil && found {
					return true
				}
			}
		}

		return false
	}

	danglingFilter := func(image *Image) bool {
		if opts.Filters.Include("dangling") {
			if len(image.RepoTags) == 0 {
				image.RepoTags = []string{"<none>:<none>"}
			}

			if opts.Filters.ExactMatch("dangling", "true") && image.RepoTags[0] != "<none>:<none>" {
				// for dangling true, filter out non-dangling images
				return false
			}

			if opts.Filters.ExactMatch("dangling", "false") && image.RepoTags[0] == "<none>:<none>" {
				// for dangling false, filter out dangling images
				return false
			}
		}
		return true
	}

	filtered := make([]*Image, 0, len(images))
	for _, image := range images {
		if includeAll(image) && includeFilter(image) && referenceFilter(image, "reference") && danglingFilter(image) {
			filtered = append(filtered, image)
		}
	}
	return filtered
}
