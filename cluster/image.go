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

	repo := ""
	name := IDOrName
	if strings.Contains(IDOrName, "/") {
		parts := strings.SplitN(IDOrName, "/", 2)
		repo = parts[0] + "/"
		name = parts[1]
	}
	if matchTag {
		if len(strings.SplitN(name, ":", 2)) == 1 {
			name = repo + name + ":latest"
		} else {
			name = repo + name
		}
	} else {
		name = repo + strings.SplitN(name, ":", 2)[0]
	}

	for _, repoTag := range image.RepoTags {
		if matchTag == false {
			if strings.Contains(repoTag, "/") {
				parts := strings.SplitN(repoTag, "/", 2)
				repoTag = parts[0] + "/" + strings.SplitN(parts[1], ":", 2)[0]
			} else {
				repoTag = strings.SplitN(repoTag, ":", 2)[0]
			}
		}
		if repoTag == name {
			return true
		}
	}
	return false
}
