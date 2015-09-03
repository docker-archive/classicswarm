package cluster

import (
	"io"

	"github.com/samalba/dockerclient"
)

// Cluster is exported
type Cluster interface {
	// Create a container
	CreateContainer(config *ContainerConfig, name string) (*Container, error)

	// Remove a container
	RemoveContainer(container *Container, force bool) error

	// Return all images
	Images(all bool) []*Image

	// Return one image matching `IDOrName`
	Image(IDOrName string) *Image

	// Remove images from the cluster
	RemoveImages(name string, force bool) ([]*dockerclient.ImageDelete, error)

	// Return all containers
	Containers() Containers

	// Return container the matching `IDOrName`
	// TODO: remove this method from the interface as we can use
	// cluster.Containers().Get(IDOrName)
	Container(IDOrName string) *Container

	// Pull images
	// `callback` can be called multiple time
	//  `where` is where it is being pulled
	//  `status` is the current status, like "", "in progress" or "downloaded
	Pull(name string, authConfig *dockerclient.AuthConfig, callback func(where, status string))

	// Import image
	// `callback` can be called multiple time
	// `where` is where it is being imported
	// `status` is the current status, like "", "in progress" or "imported"
	Import(source string, repository string, tag string, imageReader io.Reader, callback func(where, status string))

	// Load images
	// `callback` can be called multiple time
	// `what` is what is being loaded
	// `status` is the current status, like "", "in progress" or "loaded"
	Load(imageReader io.Reader, callback func(what, status string))

	// Return some info about the cluster, like nb or containers / images
	// It is pretty open, so the implementation decides what to return.
	Info() [][]string

	// Return the total memory of the cluster
	TotalMemory() int64

	// Return the number of CPUs in the cluster
	TotalCpus() int64

	// Register an event handler for cluster-wide events.
	RegisterEventHandler(h EventHandler) error

	// FIXME: remove this method
	// Return a random engine
	RANDOMENGINE() (*Engine, error)

	// RenameContainer rename a container
	RenameContainer(container *Container, newName string) error

	// BuildImage build an image
	BuildImage(*dockerclient.BuildImage, io.Writer) error

	// TagImage tag an image
	TagImage(IDOrName string, repo string, tag string, force bool) error
}
