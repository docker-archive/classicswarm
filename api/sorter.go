package api

import (
	"github.com/docker/engine-api/types"
	"github.com/docker/swarm/cluster"
)

// ContainerSorter implements the Sort interface to sort Docker containers.
// It is not guaranteed to be a stable sort.
type ContainerSorter []*cluster.Container

// Len returns the number of containers to be sorted.
func (s ContainerSorter) Len() int {
	return len(s)
}

// Swap exchanges the container elements with indices i and j.
func (s ContainerSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less reports whether the container with index i should sort before the container with index j.
// Containers are sorted chronologically by when they were created.
func (s ContainerSorter) Less(i, j int) bool {
	return s[i].Info.Created < s[j].Info.Created
}

// ImageSorter implements the Sort interface to sort Docker Images.
// It is not guaranteed to be a stable sort.
type ImageSorter []types.Image

// Len returns the number of images to be sorted.
func (s ImageSorter) Len() int {
	return len(s)
}

// Swap exchanges the container elements with indices i and j.
func (s ImageSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less reports whether the Image with index i should sort before the Image with index j.
// Images are sorted chronologically by when they were created.
func (s ImageSorter) Less(i, j int) bool {
	return s[i].Created < s[j].Created
}
