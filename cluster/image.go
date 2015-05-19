package cluster

import (
	"strings"

	"github.com/samalba/dockerclient"
)

// Image is exported
type Image struct {
	dockerclient.Image

	Engine *Engine
}

// Match is exported
func (image *Image) Match(IDOrName string) bool {
	size := len(IDOrName)

	if image.Id == IDOrName || (size > 2 && strings.HasPrefix(image.Id, IDOrName)) {
		return true
	}

	if len(strings.SplitN(IDOrName, ":", 2)) == 1 {
		IDOrName = IDOrName + ":latest"
	}

	for _, repoTag := range image.RepoTags {
		if repoTag == IDOrName {
			return true
		}
	}
	return false
}

// MatchWithoutTag is exported
func (image *Image) MatchWithoutTag(IDOrName string) bool {
	size := len(IDOrName)

	if image.Id == IDOrName || (size > 2 && strings.HasPrefix(image.Id, IDOrName)) {
		return true
	}

	name := strings.SplitN(IDOrName, ":", 2)[0]

	for _, repoTag := range image.RepoTags {
		repoName := strings.SplitN(repoTag, ":", 2)[0]
		if repoName == name {
			return true
		}
	}
	return false
}
