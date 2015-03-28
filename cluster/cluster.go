package cluster

import "github.com/samalba/dockerclient"

// Cluster is exported
type Cluster interface {
	// Create a container
	CreateContainer(config *dockerclient.ContainerConfig, name string) (*Container, error)

	// Remove a container
	RemoveContainer(container *Container, force bool) error

	// Return all images
	Images() []*Image

	// Return one image matching `IDOrName`
	Image(IDOrName string) *Image

	// Remove an image from the cluster
	RemoveImage(image *Image) ([]*dockerclient.ImageDelete, error)

	// Return all containers
	Containers() []*Container

	// Return container the matching `IDOrName`
	Container(IDOrName string) *Container

	// Pull images
	// `callback` can be called multiple time
	//  `what` is what is being pulled
	//  `status` is the current status, like "", "in progress" or "downloaded
	Pull(name string, callback func(what, status string))

	// Return some info about the cluster, like nb or containers / images
	// It is pretty open, so the implementation decides what to return.
	Info() [][2]string
}
