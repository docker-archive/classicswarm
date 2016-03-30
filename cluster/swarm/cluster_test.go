package swarm

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/docker/engine-api/types"
	engineapimock "github.com/docker/swarm/api/mockclient"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"github.com/samalba/dockerclient/mockclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type nopCloser struct {
	io.Reader
}

// Close
func (nopCloser) Close() error {
	return nil
}

var (
	mockInfo = types.Info{
		ID:              "test-engine",
		Name:            "name",
		NCPU:            10,
		MemTotal:        20,
		Driver:          "driver-test",
		ExecutionDriver: "execution-driver-test",
		KernelVersion:   "1.2.3",
		OperatingSystem: "golang",
		Labels:          []string{"foo=bar"},
	}

	mockVersion = types.Version{
		Version: "1.6.2",
	}

	engOpts = &cluster.EngineOpts{
		RefreshMinInterval: time.Duration(30) * time.Second,
		RefreshMaxInterval: time.Duration(60) * time.Second,
		FailureRetry:       3,
	}
)

func createEngine(t *testing.T, ID string, containers ...*cluster.Container) *cluster.Engine {
	engine := cluster.NewEngine(ID, 0, engOpts)
	engine.Name = ID
	engine.ID = ID

	for _, container := range containers {
		container.Engine = engine
		engine.AddContainer(container)
	}

	return engine
}

func TestContainerLookup(t *testing.T) {
	c := &Cluster{
		engines: make(map[string]*cluster.Engine),
	}
	container1 := &cluster.Container{
		Container: dockerclient.Container{
			Id:    "container1-id",
			Names: []string{"/container1-name1", "/container1-name2"},
		},
		Config: cluster.BuildContainerConfig(dockerclient.ContainerConfig{
			Labels: map[string]string{
				"com.docker.swarm.id": "swarm1-id",
			},
		}),
	}

	container2 := &cluster.Container{
		Container: dockerclient.Container{
			Id:    "container2-id",
			Names: []string{"/con"},
		},
		Config: cluster.BuildContainerConfig(dockerclient.ContainerConfig{
			Labels: map[string]string{
				"com.docker.swarm.id": "swarm2-id",
			},
		}),
	}

	n := createEngine(t, "test-engine", container1, container2)
	c.engines[n.ID] = n

	assert.Equal(t, len(c.Containers()), 2)

	// Invalid lookup
	assert.Nil(t, c.Container("invalid-id"))
	assert.Nil(t, c.Container(""))
	// Container ID lookup.
	assert.NotNil(t, c.Container("container1-id"))
	// Container ID prefix lookup.
	assert.NotNil(t, c.Container("container1-"))
	assert.Nil(t, c.Container("container"))
	// Container name lookup.
	assert.NotNil(t, c.Container("container1-name1"))
	assert.NotNil(t, c.Container("container1-name2"))
	// Container engine/name matching.
	assert.NotNil(t, c.Container("test-engine/container1-name1"))
	assert.NotNil(t, c.Container("test-engine/container1-name2"))
	// Swarm ID lookup.
	assert.NotNil(t, c.Container("swarm1-id"))
	// Swarm ID prefix lookup.
	assert.NotNil(t, c.Container("swarm1-"))
	assert.Nil(t, c.Container("swarm"))
	// Match name before ID prefix
	cc := c.Container("con")
	assert.NotNil(t, cc)
	assert.Equal(t, cc.Id, "container2-id")
}

func TestImportImage(t *testing.T) {
	// create cluster
	c := &Cluster{
		engines: make(map[string]*cluster.Engine),
	}

	// create engione
	id := "test-engine"
	engine := cluster.NewEngine(id, 0, engOpts)
	engine.Name = id
	engine.ID = id

	// create mock client
	client := mockclient.NewMockClient()
	apiClient := engineapimock.NewMockClient()
	apiClient.On("Info", mock.Anything).Return(mockInfo, nil)
	apiClient.On("ServerVersion", mock.Anything).Return(mockVersion, nil)
	apiClient.On("NetworkList", mock.Anything,
		mock.AnythingOfType("NetworkListOptions"),
	).Return([]types.NetworkResource{}, nil)
	apiClient.On("VolumeList", mock.Anything, mock.Anything).Return(types.VolumesListResponse{}, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{}, nil).Once()
	apiClient.On("ImageList", mock.Anything, mock.AnythingOfType("ImageListOptions")).Return([]types.Image{}, nil)

	// connect client
	engine.ConnectWithClient(client, apiClient)

	// add engine to cluster
	c.engines[engine.ID] = engine

	// import success
	readCloser := nopCloser{bytes.NewBufferString("ok")}
	client.On("ImportImage", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("*io.PipeReader")).Return(readCloser, nil).Once()

	callback := func(what, status string, err error) {
		// import success
		assert.Nil(t, err)
	}
	c.Import("-", "testImageOK", "latest", bytes.NewReader(nil), callback)

	// import error
	readCloser = nopCloser{bytes.NewBufferString("error")}
	err := fmt.Errorf("Import error")
	client.On("ImportImage", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("*io.PipeReader")).Return(readCloser, err).Once()

	callback = func(what, status string, err error) {
		// import error
		assert.NotNil(t, err)
	}
	c.Import("-", "testImageError", "latest", bytes.NewReader(nil), callback)
}

func TestLoadImage(t *testing.T) {
	// create cluster
	c := &Cluster{
		engines: make(map[string]*cluster.Engine),
	}

	// create engione
	id := "test-engine"
	engine := cluster.NewEngine(id, 0, engOpts)
	engine.Name = id
	engine.ID = id

	// create mock client
	client := mockclient.NewMockClient()
	apiClient := engineapimock.NewMockClient()
	apiClient.On("Info", mock.Anything).Return(mockInfo, nil)
	apiClient.On("ServerVersion", mock.Anything).Return(mockVersion, nil)
	apiClient.On("NetworkList", mock.Anything,
		mock.AnythingOfType("NetworkListOptions"),
	).Return([]types.NetworkResource{}, nil)
	apiClient.On("VolumeList", mock.Anything, mock.Anything).Return(types.VolumesListResponse{}, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{}, nil).Once()
	apiClient.On("ImageList", mock.Anything, mock.AnythingOfType("ImageListOptions")).Return([]types.Image{}, nil)

	// connect client
	engine.ConnectWithClient(client, apiClient)

	// add engine to cluster
	c.engines[engine.ID] = engine

	// load success
	client.On("LoadImage", mock.AnythingOfType("*io.PipeReader")).Return(nil).Once()
	callback := func(what, status string, err error) {
		//if load OK, err will be nil
		assert.Nil(t, err)
	}
	c.Load(bytes.NewReader(nil), callback)

	// load error
	err := fmt.Errorf("Load error")
	client.On("LoadImage", mock.AnythingOfType("*io.PipeReader")).Return(err).Once()
	callback = func(what, status string, err error) {
		// load error, err is not nil
		assert.NotNil(t, err)
	}
	c.Load(bytes.NewReader(nil), callback)
}

func TestTagImage(t *testing.T) {
	// create cluster
	c := &Cluster{
		engines: make(map[string]*cluster.Engine),
	}
	images := []types.Image{}

	image1 := types.Image{
		ID:       "1234567890",
		RepoTags: []string{"busybox:latest"},
	}
	images = append(images, image1)

	// create engine
	id := "test-engine"
	engine := cluster.NewEngine(id, 0, engOpts)
	engine.Name = id
	engine.ID = id

	// create mock client
	client := mockclient.NewMockClient()
	apiClient := engineapimock.NewMockClient()
	apiClient.On("Info", mock.Anything).Return(mockInfo, nil)
	apiClient.On("ServerVersion", mock.Anything).Return(mockVersion, nil)
	apiClient.On("NetworkList", mock.Anything,
		mock.AnythingOfType("NetworkListOptions"),
	).Return([]types.NetworkResource{}, nil)
	apiClient.On("VolumeList", mock.Anything, mock.Anything).Return(types.VolumesListResponse{}, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{}, nil).Once()
	apiClient.On("ImageList", mock.Anything, mock.AnythingOfType("ImageListOptions")).Return(images, nil)

	// connect client
	engine.ConnectWithClient(client, apiClient)

	// add engine to cluster
	c.engines[engine.ID] = engine

	// tag image
	client.On("TagImage", mock.Anything, mock.Anything, mock.Anything, false).Return(nil).Once()
	assert.Nil(t, c.TagImage("busybox", "test_busybox", "latest", false))
	assert.NotNil(t, c.TagImage("busybox_not_exists", "test_busybox", "latest", false))
}
