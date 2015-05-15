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
	for _, repoTag := range image.RepoTags {
		if len(strings.SplitN(repoTag, ":", 2)) == 1 {
			repoTag = repoTag + ":latest"
		}
		if repoTag == IDOrName {
			return true
		}
	}
	return false
}
