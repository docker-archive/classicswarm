package api

import (
	"github.com/samalba/dockerclient"
)

// ContainerSorter implements the Sort interface to sort Docker containers.
// It is not guaranteed to be a stable sort.
type ContainerSorter []*dockerclient.Container

// Len returns the number of containers to be sorted.
func (s ContainerSorter) Len() int {
	return len(s)
}

// Swap exchanges the container elements with indices i and j.
func (s ContainerSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less reports whether the container with index i should sort before the container with index j.
func (s ContainerSorter) Less(i, j int) bool {
	return s[i].Created < s[j].Created
}
