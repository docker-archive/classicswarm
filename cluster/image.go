package cluster

import (
	"strings"

	"github.com/docker/docker/pkg/parsers"
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
