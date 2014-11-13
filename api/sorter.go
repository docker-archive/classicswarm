package api

import (
	"github.com/samalba/dockerclient"
)

type ContainerSorter []*dockerclient.Container

func (s ContainerSorter) Len() int {
	return len(s)
}

func (s ContainerSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ContainerSorter) Less(i, j int) bool {
	return s[i].Created < s[j].Created
}
