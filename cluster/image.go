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

func toImageName(repo string, name string, tag string) string {
	fullname := name
	if tag != "" {
		fullname = name + ":" + tag
	}
	if repo != "" {
		fullname = repo + "/" + fullname
	}
	return fullname
}

func parseImageName(fullname string) (repo string, name string, tag string) {
	parts := strings.SplitN(fullname, "/", 2)

	nameAndTag := parts[0]
	if len(parts) == 2 {
		repo = parts[0]
		nameAndTag = parts[1]
	}

	parts = strings.SplitN(nameAndTag, ":", 2)
	name = parts[0]
	if len(parts) == 2 {
		tag = parts[1]
	}

	return
}

// Match is exported
func (image *Image) Match(IDOrName string, matchTag bool) bool {
	size := len(IDOrName)

	if image.Id == IDOrName || (size > 2 && strings.HasPrefix(image.Id, IDOrName)) {
		return true
	}

	imageName := IDOrName
	repo, name, tag := parseImageName(imageName)
	if matchTag {
		if tag == "" {
			imageName = toImageName(repo, name, "latest")
		}
	} else {
		imageName = toImageName(repo, name, "")
	}

	for _, repoTag := range image.RepoTags {
		if matchTag == false {
			r, n, _ := parseImageName(repoTag)
			repoTag = toImageName(r, n, "")
		}
		if repoTag == imageName {
			return true
		}
	}
	return false
}
