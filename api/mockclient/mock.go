package mockclient

import (
	"io"

	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/engine-api/types/network"
	"github.com/docker/engine-api/types/registry"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

// MockClient is a mock API Client based on engine-api
type MockClient struct {
	mock.Mock
}

// NewMockClient creates a new mock client
func NewMockClient() *MockClient {
	return &MockClient{}
}

// ClientVersion returns the version string associated with this instance of the Client
func (client *MockClient) ClientVersion() string {
	args := client.Mock.Called()
	return args.String(0)
}

// CheckpointCreate creates a checkpoint from the given container with the given name
func (client *MockClient) CheckpointCreate(ctx context.Context, container string, options types.CheckpointCreateOptions) error {
	args := client.Mock.Called(ctx, container, options)
	return args.Error(0)
}

// CheckpointDelete deletes the checkpoint with the given name from the given container
func (client *MockClient) CheckpointDelete(ctx context.Context, container string, checkpointID string) error {
	args := client.Mock.Called(ctx, container, checkpointID)
	return args.Error(0)
}

// CheckpointList returns the volumes configured in the docker host
func (client *MockClient) CheckpointList(ctx context.Context, container string) ([]types.Checkpoint, error) {
	args := client.Mock.Called(ctx, container)
	return args.Get(0).([]types.Checkpoint), args.Error(1)
}

// ContainerAttach attaches a connection to a container in the server
func (client *MockClient) ContainerAttach(ctx context.Context, container string, options types.ContainerAttachOptions) (types.HijackedResponse, error) {
	args := client.Mock.Called(ctx, container, options)
	return args.Get(0).(types.HijackedResponse), args.Error(1)
}

// ContainerCommit applies changes into a container and creates a new tagged image
func (client *MockClient) ContainerCommit(ctx context.Context, container string, options types.ContainerCommitOptions) (types.ContainerCommitResponse, error) {
	args := client.Mock.Called(ctx, container, options)
	return args.Get(0).(types.ContainerCommitResponse), args.Error(1)
}

// ContainerCreate creates a new container based in the given configuration
func (client *MockClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (types.ContainerCreateResponse, error) {
	args := client.Mock.Called(ctx, config, hostConfig, networkingConfig, containerName)
	return args.Get(0).(types.ContainerCreateResponse), args.Error(1)
}

// ContainerDiff shows differences in a container filesystem since it was started
func (client *MockClient) ContainerDiff(ctx context.Context, container string) ([]types.ContainerChange, error) {
	args := client.Mock.Called(ctx, container)
	return args.Get(0).([]types.ContainerChange), args.Error(1)
}

// ContainerExecAttach attaches a connection to an exec process in the server
func (client *MockClient) ContainerExecAttach(ctx context.Context, execID string, config types.ExecConfig) (types.HijackedResponse, error) {
	args := client.Mock.Called(ctx, execID, config)
	return args.Get(0).(types.HijackedResponse), args.Error(1)
}

// ContainerExecCreate creates a new exec configuration to run an exec process
func (client *MockClient) ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.ContainerExecCreateResponse, error) {
	args := client.Mock.Called(ctx, container, config)
	return args.Get(0).(types.ContainerExecCreateResponse), args.Error(1)
}

// ContainerExecInspect returns information about a specific exec process on the docker host
func (client *MockClient) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	args := client.Mock.Called(ctx, execID)
	return args.Get(0).(types.ContainerExecInspect), args.Error(1)
}

// ContainerExecResize changes the size of the tty for an exec process running inside a container
func (client *MockClient) ContainerExecResize(ctx context.Context, execID string, options types.ResizeOptions) error {
	args := client.Mock.Called(ctx, execID, options)
	return args.Error(0)
}

// ContainerExecStart starts an exec process already create in the docker host
func (client *MockClient) ContainerExecStart(ctx context.Context, execID string, config types.ExecStartCheck) error {
	args := client.Mock.Called(ctx, execID, config)
	return args.Error(0)
}

// ContainerExport retrieves the raw contents of a container and returns them as an io.ReadCloser
func (client *MockClient) ContainerExport(ctx context.Context, container string) (io.ReadCloser, error) {
	args := client.Mock.Called(ctx, container)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// ContainerInspect returns the container information
func (client *MockClient) ContainerInspect(ctx context.Context, container string) (types.ContainerJSON, error) {
	args := client.Mock.Called(ctx, container)
	return args.Get(0).(types.ContainerJSON), args.Error(1)
}

// ContainerInspectWithRaw returns the container information and its raw representation
func (client *MockClient) ContainerInspectWithRaw(ctx context.Context, container string, getSize bool) (types.ContainerJSON, []byte, error) {
	args := client.Mock.Called(ctx, container, getSize)
	return args.Get(0).(types.ContainerJSON), args.Get(1).([]byte), args.Error(2)
}

// ContainerKill terminates the container process but does not remove the container from the docker host
func (client *MockClient) ContainerKill(ctx context.Context, container, signal string) error {
	args := client.Mock.Called(ctx, container, signal)
	return args.Error(0)
}

// ContainerList returns the list of containers in the docker host
func (client *MockClient) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	args := client.Mock.Called(ctx, options)
	return args.Get(0).([]types.Container), args.Error(1)
}

// ContainerLogs returns the logs generated by a container in an io.ReadCloser
func (client *MockClient) ContainerLogs(ctx context.Context, container string, options types.ContainerLogsOptions) (io.ReadCloser, error) {
	args := client.Mock.Called(ctx, container, options)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// ContainerPause pauses the main process of a given container without terminating it
func (client *MockClient) ContainerPause(ctx context.Context, container string) error {
	args := client.Mock.Called(ctx, container)
	return args.Error(0)
}

// ContainerRemove kills and removes a container from the docker host
func (client *MockClient) ContainerRemove(ctx context.Context, container string, options types.ContainerRemoveOptions) error {
	args := client.Mock.Called(ctx, container, options)
	return args.Error(0)
}

// ContainerRename changes the name of a given container
func (client *MockClient) ContainerRename(ctx context.Context, container, newContainerName string) error {
	args := client.Mock.Called(ctx, container, newContainerName)
	return args.Error(0)
}

// ContainerResize changes the size of the tty for a container
func (client *MockClient) ContainerResize(ctx context.Context, container string, options types.ResizeOptions) error {
	args := client.Mock.Called(ctx, container, options)
	return args.Error(0)
}

// ContainerRestart stops and starts a container again
func (client *MockClient) ContainerRestart(ctx context.Context, container string, timeout int) error {
	args := client.Mock.Called(ctx, container, timeout)
	return args.Error(0)
}

// ContainerStatPath returns Stat information about a path inside the container filesystem
func (client *MockClient) ContainerStatPath(ctx context.Context, container, path string) (types.ContainerPathStat, error) {
	args := client.Mock.Called(ctx, container, path)
	return args.Get(0).(types.ContainerPathStat), args.Error(1)
}

// ContainerStats returns near realtime stats for a given container
func (client *MockClient) ContainerStats(ctx context.Context, container string, stream bool) (io.ReadCloser, error) {
	args := client.Mock.Called(ctx, container, stream)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// ContainerStart sends a request to the docker daemon to start a container
func (client *MockClient) ContainerStart(ctx context.Context, container string, checkpointID string) error {
	args := client.Mock.Called(ctx, container, checkpointID)
	return args.Error(0)
}

// ContainerStop stops a container without terminating the process
func (client *MockClient) ContainerStop(ctx context.Context, container string, timeout int) error {
	args := client.Mock.Called(ctx, container, timeout)
	return args.Error(0)
}

// ContainerTop shows process information from within a container
func (client *MockClient) ContainerTop(ctx context.Context, container string, arguments []string) (types.ContainerProcessList, error) {
	args := client.Mock.Called(ctx, container, arguments)
	return args.Get(0).(types.ContainerProcessList), args.Error(1)
}

// ContainerUnpause resumes the process execution within a container
func (client *MockClient) ContainerUnpause(ctx context.Context, container string) error {
	args := client.Mock.Called(ctx, container)
	return args.Error(0)
}

// ContainerUpdate updates resources of a container
func (client *MockClient) ContainerUpdate(ctx context.Context, container string, updateConfig container.UpdateConfig) error {
	args := client.Mock.Called(ctx, container, updateConfig)
	return args.Error(0)
}

// ContainerWait pauses execution until a container exits
func (client *MockClient) ContainerWait(ctx context.Context, container string) (int, error) {
	args := client.Mock.Called(ctx, container)
	return args.Int(0), args.Error(1)
}

// CopyFromContainer gets the content from the container and returns it as a Reader to manipulate it in the host
func (client *MockClient) CopyFromContainer(ctx context.Context, container, srcPath string) (io.ReadCloser, types.ContainerPathStat, error) {
	args := client.Mock.Called(ctx, container, srcPath)
	return args.Get(0).(io.ReadCloser), args.Get(1).(types.ContainerPathStat), args.Error(2)
}

// CopyToContainer copies content into the container filesystem
func (client *MockClient) CopyToContainer(ctx context.Context, container, path string, content io.Reader, options types.CopyToContainerOptions) error {
	args := client.Mock.Called(ctx, container, path, content, options)
	return args.Error(0)
}

// Events returns a stream of events in the daemon in a ReadCloser
func (client *MockClient) Events(ctx context.Context, options types.EventsOptions) (io.ReadCloser, error) {
	args := client.Mock.Called(ctx, options)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// ImageBuild sends request to the daemon to build images
func (client *MockClient) ImageBuild(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	args := client.Mock.Called(ctx, context, options)
	return args.Get(0).(types.ImageBuildResponse), args.Error(1)
}

// ImageCreate creates a new image based in the parent options
func (client *MockClient) ImageCreate(ctx context.Context, parentReference string, options types.ImageCreateOptions) (io.ReadCloser, error) {
	args := client.Mock.Called(ctx, parentReference, options)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// ImageHistory returns the changes in an image in history format
func (client *MockClient) ImageHistory(ctx context.Context, image string) ([]types.ImageHistory, error) {
	args := client.Mock.Called(ctx, image)
	return args.Get(0).([]types.ImageHistory), args.Error(1)
}

// ImageImport creates a new image based in the source options
func (client *MockClient) ImageImport(ctx context.Context, source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error) {
	args := client.Mock.Called(ctx, source, ref, options)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// ImageInspectWithRaw returns the image information and it's raw representation
func (client *MockClient) ImageInspectWithRaw(ctx context.Context, image string, getSize bool) (types.ImageInspect, []byte, error) {
	args := client.Mock.Called(ctx, image, getSize)
	return args.Get(0).(types.ImageInspect), args.Get(1).([]byte), args.Error(2)
}

// ImageList returns a list of images in the docker host
func (client *MockClient) ImageList(ctx context.Context, options types.ImageListOptions) ([]types.Image, error) {
	args := client.Mock.Called(ctx, options)
	return args.Get(0).([]types.Image), args.Error(1)
}

// ImageLoad loads an image in the docker host from the client host
func (client *MockClient) ImageLoad(ctx context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error) {
	args := client.Mock.Called(ctx, input, quiet)
	return args.Get(0).(types.ImageLoadResponse), args.Error(1)
}

// ImagePull requests the docker host to pull an image from a remote registry
func (client *MockClient) ImagePull(ctx context.Context, ref string, options types.ImagePullOptions) (io.ReadCloser, error) {
	args := client.Mock.Called(ctx, ref, options)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// ImagePush requests the docker host to push an image to a remote registry
func (client *MockClient) ImagePush(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error) {
	args := client.Mock.Called(ctx, ref, options)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// ImageRemove removes an image from the docker host
func (client *MockClient) ImageRemove(ctx context.Context, image string, options types.ImageRemoveOptions) ([]types.ImageDelete, error) {
	args := client.Mock.Called(ctx, image, options)
	return args.Get(0).([]types.ImageDelete), args.Error(1)
}

// ImageSearch makes the docker host to search by a term in a remote registry
func (client *MockClient) ImageSearch(ctx context.Context, term string, options types.ImageSearchOptions) ([]registry.SearchResult, error) {
	args := client.Mock.Called(ctx, term, options)
	return args.Get(0).([]registry.SearchResult), args.Error(1)
}

// ImageSave retrieves one or more images from the docker host as an io.ReadCloser
func (client *MockClient) ImageSave(ctx context.Context, images []string) (io.ReadCloser, error) {
	args := client.Mock.Called(ctx, images)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// ImageTag tags an image in the docker host
func (client *MockClient) ImageTag(ctx context.Context, image, ref string, options types.ImageTagOptions) error {
	args := client.Mock.Called(ctx, image, ref, options)
	return args.Error(0)
}

// Info returns information about the docker server
func (client *MockClient) Info(ctx context.Context) (types.Info, error) {
	args := client.Mock.Called(ctx)
	return args.Get(0).(types.Info), args.Error(1)
}

// NetworkConnect connects a container to an existent network in the docker host
func (client *MockClient) NetworkConnect(ctx context.Context, networkID, container string, config *network.EndpointSettings) error {
	args := client.Mock.Called(ctx, networkID, container, config)
	return args.Error(0)
}

// NetworkCreate creates a new network in the docker host
func (client *MockClient) NetworkCreate(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error) {
	args := client.Mock.Called(ctx, name, options)
	return args.Get(0).(types.NetworkCreateResponse), args.Error(1)
}

// NetworkDisconnect disconnects a container from an existent network in the docker host
func (client *MockClient) NetworkDisconnect(ctx context.Context, networkID, container string, force bool) error {
	args := client.Mock.Called(ctx, networkID, container, force)
	return args.Error(0)
}

// NetworkInspect returns the information for a specific network configured in the docker host
func (client *MockClient) NetworkInspect(ctx context.Context, networkID string) (types.NetworkResource, error) {
	args := client.Mock.Called(ctx, networkID)
	return args.Get(0).(types.NetworkResource), args.Error(1)
}

// NetworkInspectWithRaw returns the information for a specific network configured in the docker host and it's raw representation
func (client *MockClient) NetworkInspectWithRaw(ctx context.Context, networkID string) (types.NetworkResource, []byte, error) {
	args := client.Mock.Called(ctx, networkID)
	return args.Get(0).(types.NetworkResource), args.Get(1).([]byte), args.Error(2)
}

// NetworkList returns the list of networks configured in the docker host
func (client *MockClient) NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
	args := client.Mock.Called(ctx, options)
	return args.Get(0).([]types.NetworkResource), args.Error(1)
}

// NetworkRemove removes an existent network from the docker host
func (client *MockClient) NetworkRemove(ctx context.Context, networkID string) error {
	args := client.Mock.Called(ctx, networkID)
	return args.Error(0)
}

// RegistryLogin authenticates the docker server with a given docker registry
func (client *MockClient) RegistryLogin(ctx context.Context, auth types.AuthConfig) (types.AuthResponse, error) {
	args := client.Mock.Called(ctx, auth)
	return args.Get(0).(types.AuthResponse), args.Error(1)
}

// ServerVersion returns information of the docker client and server host
func (client *MockClient) ServerVersion(ctx context.Context) (types.Version, error) {
	args := client.Mock.Called(ctx)
	return args.Get(0).(types.Version), args.Error(1)
}

// UpdateClientVersion updates the client version
func (client *MockClient) UpdateClientVersion(v string) {
}

// VolumeCreate creates a volume in the docker host
func (client *MockClient) VolumeCreate(ctx context.Context, options types.VolumeCreateRequest) (types.Volume, error) {
	args := client.Mock.Called(ctx, options)
	return args.Get(0).(types.Volume), args.Error(1)
}

// VolumeInspect returns the information about a specific volume in the docker host
func (client *MockClient) VolumeInspect(ctx context.Context, volumeID string) (types.Volume, error) {
	args := client.Mock.Called(ctx, volumeID)
	return args.Get(0).(types.Volume), args.Error(1)
}

// VolumeInspectWithRaw returns the information about a specific volume in the docker host and its raw representation
func (client *MockClient) VolumeInspectWithRaw(ctx context.Context, volumeID string) (types.Volume, []byte, error) {
	args := client.Mock.Called(ctx, volumeID)
	return args.Get(0).(types.Volume), args.Get(1).([]byte), args.Error(2)
}

// VolumeList returns the volumes configured in the docker host
func (client *MockClient) VolumeList(ctx context.Context, filter filters.Args) (types.VolumesListResponse, error) {
	args := client.Mock.Called(ctx, filter)
	return args.Get(0).(types.VolumesListResponse), args.Error(1)
}

// VolumeRemove removes a volume from the docker host
func (client *MockClient) VolumeRemove(ctx context.Context, volumeID string) error {
	args := client.Mock.Called(ctx, volumeID)
	return args.Error(0)
}
