package cluster

import (
	"strings"

	"github.com/samalba/dockerclient"
)

type Image struct {
	dockerclient.Image

	Node Node
}

func (image *Image) Match(IdOrName string) bool {
	size := len(IdOrName)

	if image.Id == IdOrName || (size > 2 && strings.HasPrefix(image.Id, IdOrName)) {
		return true
	}
	for _, repoTag := range image.RepoTags {
		if repoTag == IdOrName || (size > 2 && strings.HasPrefix(repoTag, IdOrName)) {
			return true
		}
	}
	return false
}
