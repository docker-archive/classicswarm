package cluster

import (
	"io"

	"github.com/docker/engine-api/types"
	"github.com/samalba/dockerclient"
)

// Cluster is exported
type Cluster interface {
	// Create a container
	CreateContainer(config *ContainerConfig, name string, authConfig *dockerclient.AuthConfig) (*Container, error)

	// Remove a container
	RemoveContainer(container *Container, force, volumes bool) error

	// Return all images
	Images() Images

	// Return one image matching `IDOrName`
	Image(IDOrName string) *Image

	// Remove images from the cluster
	RemoveImages(name string, force bool) ([]types.ImageDelete, error)

	// Return all containers
	Containers() Containers

	// Start a container
	StartContainer(container *Container, hostConfig *dockerclient.HostConfig) error

	// Return container the matching `IDOrName`
	// TODO: remove this method from the interface as we can use
	// cluster.Containers().Get(IDOrName)
	Container(IDOrName string) *Container

	// Return all networks
	Networks() Networks

	// Create a network
	CreateNetwork(request *types.NetworkCreate) (*types.NetworkCreateResponse, error)

	// Remove a network from the cluster
	RemoveNetwork(network *Network) error

	// Create a volume
	CreateVolume(request *types.VolumeCreateRequest) (*Volume, error)

	// Return all volumes
	Volumes() Volumes

	// Remove volumes from the cluster
	RemoveVolumes(name string) (bool, error)

	// Pull images
	// `callback` can be called multiple time
	//  `where` is where it is being pulled
	//  `status` is the current status, like "", "in progress" or "downloaded
	Pull(name string, authConfig *dockerclient.AuthConfig, callback func(where, status string, err error))

	// Import image
	// `callback` can be called multiple time
	// `where` is where it is being imported
	// `status` is the current status, like "", "in progress" or "imported"
	Import(source string, repository string, tag string, imageReader io.Reader, callback func(where, status string, err error))

	// Load images
	// `callback` can be called multiple time
	// `what` is what is being loaded
	// `status` is the current status, like "", "in progress" or "loaded"
	Load(imageReader io.Reader, callback func(what, status string, err error))

	// Return some info about the cluster, like nb of containers / images
	// It is pretty open, so the implementation decides what to return.
	Info() [][2]string

	// Return the total memory of the cluster
	TotalMemory() int64

	// Return the number of CPUs in the cluster
	TotalCpus() int

	// Register an event handler for cluster-wide events.
	RegisterEventHandler(h EventHandler) error

	// Unregister an event handler.
	UnregisterEventHandler(h EventHandler)

	// FIXME: remove this method
	// Return a random engine
	RANDOMENGINE() (*Engine, error)

	// Rename a container
	RenameContainer(container *Container, newName string) error

	// Build an image
	BuildImage(*types.ImageBuildOptions, io.Writer) error

	// Tag an image
	TagImage(IDOrName string, repo string, tag string, force bool) error
}
