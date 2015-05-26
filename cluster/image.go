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
func (image *Image) Match(IDOrName string, matchTag bool) bool {
	size := len(IDOrName)

	if image.Id == IDOrName || (size > 2 && strings.HasPrefix(image.Id, IDOrName)) {
		return true
	}

	name := IDOrName
	if matchTag {
		if len(strings.SplitN(IDOrName, ":", 2)) == 1 {
			name = IDOrName + ":latest"
		}
	} else {
		name = strings.SplitN(IDOrName, ":", 2)[0]
	}

	for _, repoTag := range image.RepoTags {
		if matchTag == false {
			repoTag = strings.SplitN(repoTag, ":", 2)[0]
		}
		if repoTag == name {
			return true
		}
	}
	return false
}
