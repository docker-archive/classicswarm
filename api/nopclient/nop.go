package nopclient

import (
	"errors"
	"io"
	"time"

	"golang.org/x/net/context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/api/types/volume"
)

var (
	errNoEngine = errors.New("Engine no longer exists")
)

// NopClient is a nop API Client based on docker/api
type NopClient struct {
}

// NewNopClient creates a new nop client
func NewNopClient() *NopClient {
	return &NopClient{}
}

// BuildCachePrune requests the daemon to delete unused cache data
func (client *NopClient) BuildCachePrune(ctx context.Context) (*types.BuildCachePruneReport, error) {
	return nil, errNoEngine
}

// ClientVersion returns the version string associated with this instance of the Client
func (client *NopClient) ClientVersion() string {
	return ""
}

// CheckpointCreate creates a checkpoint from the given container with the given name
func (client *NopClient) CheckpointCreate(ctx context.Context, container string, options types.CheckpointCreateOptions) error {
	return errNoEngine
}

// CheckpointDelete deletes the checkpoint with the given name from the given container
func (client *NopClient) CheckpointDelete(ctx context.Context, container string, checkpointID string) error {
	return errNoEngine
}

// CheckpointList returns the volumes configured in the docker host
func (client *NopClient) CheckpointList(ctx context.Context, container string) ([]types.Checkpoint, error) {
	return nil, errNoEngine
}

// ContainerAttach attaches a connection to a container in the server
func (client *NopClient) ContainerAttach(ctx context.Context, container string, options types.ContainerAttachOptions) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, errNoEngine
}

// ContainerCommit applies changes into a container and creates a new tagged image
func (client *NopClient) ContainerCommit(ctx context.Context, container string, options types.ContainerCommitOptions) (types.IDResponse, error) {
	return types.IDResponse{}, errNoEngine
}

// ContainerCreate creates a new container based in the given configuration
func (client *NopClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
	return container.ContainerCreateCreatedBody{}, errNoEngine
}

// ContainerDiff shows differences in a container filesystem since it was started
func (client *NopClient) ContainerDiff(ctx context.Context, container string) ([]container.ContainerChangeResponseItem, error) {
	return nil, errNoEngine
}

// ContainerExecAttach attaches a connection to an exec process in the server
func (client *NopClient) ContainerExecAttach(ctx context.Context, execID string, config types.ExecConfig) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, errNoEngine
}

// ContainerExecCreate creates a new exec configuration to run an exec process
func (client *NopClient) ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error) {
	return types.IDResponse{}, errNoEngine
}

// ContainerExecInspect returns information about a specific exec process on the docker host
func (client *NopClient) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	return types.ContainerExecInspect{}, errNoEngine
}

// ContainerExecResize changes the size of the tty for an exec process running inside a container
func (client *NopClient) ContainerExecResize(ctx context.Context, execID string, options types.ResizeOptions) error {
	return errNoEngine
}

// ContainerExecStart starts an exec process already create in the docker host
func (client *NopClient) ContainerExecStart(ctx context.Context, execID string, config types.ExecStartCheck) error {
	return errNoEngine
}

// ContainerExport retrieves the raw contents of a container and returns them as an io.ReadCloser
func (client *NopClient) ContainerExport(ctx context.Context, container string) (io.ReadCloser, error) {
	return nil, errNoEngine
}

// ContainerInspect returns the container information
func (client *NopClient) ContainerInspect(ctx context.Context, container string) (types.ContainerJSON, error) {
	return types.ContainerJSON{}, errNoEngine
}

// ContainerInspectWithRaw returns the container information and its raw representation
func (client *NopClient) ContainerInspectWithRaw(ctx context.Context, container string, getSize bool) (types.ContainerJSON, []byte, error) {
	return types.ContainerJSON{}, nil, errNoEngine
}

// ContainerKill terminates the container process but does not remove the container from the docker host
func (client *NopClient) ContainerKill(ctx context.Context, container, signal string) error {
	return errNoEngine
}

// ContainerList returns the list of containers in the docker host
func (client *NopClient) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	return nil, errNoEngine
}

// ContainerLogs returns the logs generated by a container in an io.ReadCloser
func (client *NopClient) ContainerLogs(ctx context.Context, container string, options types.ContainerLogsOptions) (io.ReadCloser, error) {
	return nil, errNoEngine
}

// ContainerPause pauses the main process of a given container without terminating it
func (client *NopClient) ContainerPause(ctx context.Context, container string) error {
	return errNoEngine
}

// ContainerRemove kills and removes a container from the docker host
func (client *NopClient) ContainerRemove(ctx context.Context, container string, options types.ContainerRemoveOptions) error {
	return errNoEngine
}

// ContainerRename changes the name of a given container
func (client *NopClient) ContainerRename(ctx context.Context, container, newContainerName string) error {
	return errNoEngine
}

// ContainerResize changes the size of the tty for a container
func (client *NopClient) ContainerResize(ctx context.Context, container string, options types.ResizeOptions) error {
	return errNoEngine
}

// ContainerRestart stops and starts a container again
func (client *NopClient) ContainerRestart(ctx context.Context, container string, timeout *time.Duration) error {
	return errNoEngine
}

// ContainerStatPath returns Stat information about a path inside the container filesystem
func (client *NopClient) ContainerStatPath(ctx context.Context, container, path string) (types.ContainerPathStat, error) {
	return types.ContainerPathStat{}, errNoEngine
}

// ContainerStats returns near realtime stats for a given container
func (client *NopClient) ContainerStats(ctx context.Context, container string, stream bool) (types.ContainerStats, error) {
	return types.ContainerStats{}, errNoEngine
}

// ContainerStart sends a request to the docker daemon to start a container
func (client *NopClient) ContainerStart(ctx context.Context, container string, options types.ContainerStartOptions) error {
	return errNoEngine
}

// ContainerStop stops a container without terminating the process
func (client *NopClient) ContainerStop(ctx context.Context, container string, timeout *time.Duration) error {
	return errNoEngine
}

// ContainerTop shows process information from within a container
func (client *NopClient) ContainerTop(ctx context.Context, ctr string, arguments []string) (container.ContainerTopOKBody, error) {
	return container.ContainerTopOKBody{}, errNoEngine
}

// ContainerUnpause resumes the process execution within a container
func (client *NopClient) ContainerUnpause(ctx context.Context, container string) error {
	return errNoEngine
}

// ContainerUpdate updates resources of a container
func (client *NopClient) ContainerUpdate(ctx context.Context, containerID string, updateConfig container.UpdateConfig) (container.ContainerUpdateOKBody, error) {
	return container.ContainerUpdateOKBody{}, errNoEngine
}

// ContainerWait pauses execution until a container exits
func (client *NopClient) ContainerWait(ctx context.Context, ctr string, condition container.WaitCondition) (<-chan container.ContainerWaitOKBody, <-chan error) {
	var errChan = make(chan error)
	errChan <- errNoEngine
	return nil, errChan
}

// CopyFromContainer gets the content from the container and returns it as a Reader to manipulate it in the host
func (client *NopClient) CopyFromContainer(ctx context.Context, container, srcPath string) (io.ReadCloser, types.ContainerPathStat, error) {
	return nil, types.ContainerPathStat{}, errNoEngine
}

// CopyToContainer copies content into the container filesystem
func (client *NopClient) CopyToContainer(ctx context.Context, container, path string, content io.Reader, options types.CopyToContainerOptions) error {
	return errNoEngine
}

// ContainersPrune requests the daemon to delete unused data
func (client *NopClient) ContainersPrune(ctx context.Context, pruneFilters filters.Args) (types.ContainersPruneReport, error) {
	return types.ContainersPruneReport{}, errNoEngine
}

// Events returns a stream of events in the daemon in a event.Message channel
func (client *NopClient) Events(ctx context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error) {
	e := make(chan events.Message)
	e <- events.Message{}
	err := make(chan error)
	err <- errNoEngine
	return e, err
}

// ImageBuild sends request to the daemon to build images
func (client *NopClient) ImageBuild(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	return types.ImageBuildResponse{}, errNoEngine
}

// ImageCreate creates a new image based in the parent options
func (client *NopClient) ImageCreate(ctx context.Context, parentReference string, options types.ImageCreateOptions) (io.ReadCloser, error) {
	return nil, errNoEngine
}

// ImageHistory returns the changes in an image in history format
func (client *NopClient) ImageHistory(ctx context.Context, image string) ([]image.HistoryResponseItem, error) {
	return nil, errNoEngine
}

// ImageImport creates a new image based in the source options
func (client *NopClient) ImageImport(ctx context.Context, source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error) {
	return nil, errNoEngine
}

// ImageInspectWithRaw returns the image information and it's raw representation
func (client *NopClient) ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error) {
	return types.ImageInspect{}, nil, errNoEngine
}

// ImageList returns a list of images in the docker host
func (client *NopClient) ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error) {
	return nil, errNoEngine
}

// ImageLoad loads an image in the docker host from the client host
func (client *NopClient) ImageLoad(ctx context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error) {
	return types.ImageLoadResponse{}, errNoEngine
}

// ImagePull requests the docker host to pull an image from a remote registry
func (client *NopClient) ImagePull(ctx context.Context, ref string, options types.ImagePullOptions) (io.ReadCloser, error) {
	return nil, errNoEngine
}

// ImagePush requests the docker host to push an image to a remote registry
func (client *NopClient) ImagePush(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error) {
	return nil, errNoEngine
}

// ImageRemove removes an image from the docker host
func (client *NopClient) ImageRemove(ctx context.Context, image string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
	return nil, errNoEngine
}

// ImageSearch makes the docker host to search by a term in a remote registry
func (client *NopClient) ImageSearch(ctx context.Context, term string, options types.ImageSearchOptions) ([]registry.SearchResult, error) {
	return nil, errNoEngine
}

// ImageSave retrieves one or more images from the docker host as an io.ReadCloser
func (client *NopClient) ImageSave(ctx context.Context, images []string) (io.ReadCloser, error) {
	return nil, errNoEngine
}

// ImageTag tags an image in the docker host
func (client *NopClient) ImageTag(ctx context.Context, image, ref string) error {
	return errNoEngine
}

// ImagesPrune requests the daemon to delete unused data
func (client *NopClient) ImagesPrune(ctx context.Context, pruneFilter filters.Args) (types.ImagesPruneReport, error) {
	return types.ImagesPruneReport{}, errNoEngine
}

// Info returns information about the docker server
func (client *NopClient) Info(ctx context.Context) (types.Info, error) {
	return types.Info{}, errNoEngine
}

// NegotiateAPIVersion updates the version string associated with this
// instance of the Client to match the latest version the server supports
func (client *NopClient) NegotiateAPIVersion(ctx context.Context) {
}

// NetworkConnect connects a container to an existent network in the docker host
func (client *NopClient) NetworkConnect(ctx context.Context, networkID, container string, config *network.EndpointSettings) error {
	return errNoEngine
}

// NetworkCreate creates a new network in the docker host
func (client *NopClient) NetworkCreate(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error) {
	return types.NetworkCreateResponse{}, errNoEngine
}

// NetworkDisconnect disconnects a container from an existent network in the docker host
func (client *NopClient) NetworkDisconnect(ctx context.Context, networkID, container string, force bool) error {
	return errNoEngine
}

// NetworkInspect returns the information for a specific network configured in the docker host
func (client *NopClient) NetworkInspect(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, error) {
	return types.NetworkResource{}, errNoEngine
}

// NetworkInspectWithRaw returns the information for a specific network configured in the docker host and it's raw representation
func (client *NopClient) NetworkInspectWithRaw(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, []byte, error) {
	return types.NetworkResource{}, nil, errNoEngine
}

// NetworkList returns the list of networks configured in the docker host
func (client *NopClient) NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
	return nil, errNoEngine
}

// NetworkRemove removes an existent network from the docker host
func (client *NopClient) NetworkRemove(ctx context.Context, networkID string) error {
	return errNoEngine
}

// NetworksPrune requests the daemon to delete unused networks
func (client *NopClient) NetworksPrune(ctx context.Context, pruneFilter filters.Args) (types.NetworksPruneReport, error) {
	return types.NetworksPruneReport{}, errNoEngine
}

// RegistryLogin authenticates the docker server with a given docker registry
func (client *NopClient) RegistryLogin(ctx context.Context, auth types.AuthConfig) (registry.AuthenticateOKBody, error) {
	return registry.AuthenticateOKBody{}, errNoEngine
}

// DiskUsage requests the current data usage from the daemon
func (client *NopClient) DiskUsage(ctx context.Context) (types.DiskUsage, error) {
	return types.DiskUsage{}, errNoEngine
}

// Ping pings the server and return the value of the "Docker-Experimental" header
func (client *NopClient) Ping(ctx context.Context) (types.Ping, error) {
	return types.Ping{}, errNoEngine
}

// ServerVersion returns information of the docker client and server host
func (client *NopClient) ServerVersion(ctx context.Context) (types.Version, error) {
	return types.Version{}, errNoEngine
}

// VolumeCreate creates a volume in the docker host
func (client *NopClient) VolumeCreate(ctx context.Context, options volume.VolumesCreateBody) (types.Volume, error) {
	return types.Volume{}, errNoEngine
}

// VolumeInspect returns the information about a specific volume in the docker host
func (client *NopClient) VolumeInspect(ctx context.Context, volumeID string) (types.Volume, error) {
	return types.Volume{}, errNoEngine
}

// VolumeInspectWithRaw returns the information about a specific volume in the docker host and its raw representation
func (client *NopClient) VolumeInspectWithRaw(ctx context.Context, volumeID string) (types.Volume, []byte, error) {
	return types.Volume{}, nil, errNoEngine
}

// VolumeList returns the volumes configured in the docker host
func (client *NopClient) VolumeList(ctx context.Context, filter filters.Args) (volume.VolumesListOKBody, error) {
	return volume.VolumesListOKBody{}, errNoEngine
}

// VolumeRemove removes a volume from the docker host
func (client *NopClient) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	return errNoEngine
}

// VolumesPrune requests the daemon to delete unused data
func (client *NopClient) VolumesPrune(ctx context.Context, pruneFilter filters.Args) (types.VolumesPruneReport, error) {
	return types.VolumesPruneReport{}, errNoEngine
}
