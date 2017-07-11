package cluster

import (
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/swarmkit/watch"
)

// Cluster is exported
type Cluster interface {
	// CreateContainer creates a container.
	CreateContainer(config *ContainerConfig, name string, authConfig *types.AuthConfig) (*Container, error)

	// RemoveContainer removes a container.
	RemoveContainer(container *Container, force, volumes bool) error

	// Images returns all images.
	Images() Images

	// Image returns one image matching `IDOrName`.
	Image(IDOrName string) *Image

	// RemoveImages removes images from the cluster.
	RemoveImages(name string, force bool) ([]types.ImageDeleteResponseItem, error)

	// Containers returns all containers.
	Containers() Containers

	// StartContainer starts a container.
	StartContainer(container *Container) error

	// Container returns the container matching `IDOrName`.
	// TODO: remove this method from the interface as we can use
	// cluster.Containers().Get(IDOrName).
	Container(IDOrName string) *Container

	// Networks returns all networks.
	Networks() Networks

	// CreateNetwork creates a network.
	CreateNetwork(name string, request *types.NetworkCreate) (*types.NetworkCreateResponse, error)

	// RemoveNetwork removes a network from the cluster.
	RemoveNetwork(network *Network) error

	// CreateVolume creates a volume.
	CreateVolume(request *volume.VolumesCreateBody) (*types.Volume, error)

	// Volumes returns all volumes.
	Volumes() Volumes

	// RemoveVolumes removes volumes from the cluster.
	RemoveVolumes(name string) (bool, error)

	// Pull images
	// `callback` can be called multiple time
	Pull(name string, authConfig *types.AuthConfig, callback func(msg JSONMessageWrapper))

	// Import image
	// `callback` can be called multiple time
	Import(source string, ref string, tag string, imageReader io.Reader, callback func(msg JSONMessageWrapper))

	// Load images
	// `callback` can be called multiple time
	Load(imageReader io.Reader, callback func(msg JSONMessageWrapper))

	// Info returns some info about the cluster, like nb of containers / images.
	// It is pretty open, so the implementation decides what to return.
	Info() [][2]string

	// TotalMemory returns the total memory of the cluster.
	TotalMemory() int64

	// TotalCpus returns the number of CPUs in the cluster.
	TotalCpus() int64

	// RegisterEventHandler registers an event handler for cluster-wide events.
	RegisterEventHandler(h EventHandler, q *watch.Queue) error

	// UnregisterEventHandler unregisters an event handler.
	UnregisterEventHandler(h EventHandler)

	// CloseWatchQueue closes the watchQueue when the manager shuts down.
	CloseWatchQueue()

	// FIXME: remove this method
	// RANDOMENGINE returns a random engine.
	RANDOMENGINE() (*Engine, error)

	// RenameContainer renames a container.
	RenameContainer(container *Container, newName string) error

	// BuildImage builds an image.
	BuildImage(io.Reader, *types.ImageBuildOptions, func(msg JSONMessageWrapper)) error

	// TagImage tags an image.
	TagImage(IDOrName string, ref string, force bool) error

	// RefreshEngine refreshes a single cluster engine.
	RefreshEngine(hostname string) error

	// RefreshEngines refreshes all engines in the cluster.
	RefreshEngines() error
}
