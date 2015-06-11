package swarm

import (
	"bytes"
	"fmt"
	"io"
	"testing"

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
	mockInfo = &dockerclient.Info{
		ID:              "id",
		Name:            "name",
		NCPU:            10,
		MemTotal:        20,
		Driver:          "driver-test",
		ExecutionDriver: "execution-driver-test",
		KernelVersion:   "1.2.3",
		OperatingSystem: "golang",
		Labels:          []string{"foo=bar"},
	}

	mockVersion = &dockerclient.Version{
		Version: "1.6.2",
	}
)

func createEngine(t *testing.T, ID string, containers ...*cluster.Container) *cluster.Engine {
	engine := cluster.NewEngine(ID, 0)
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
	engine := cluster.NewEngine(id, 0)
	engine.Name = id
	engine.ID = id

	// create mock client
	client := mockclient.NewMockClient()
	client.On("Info").Return(mockInfo, nil)
	client.On("Version").Return(mockVersion, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{}, nil).Once()
	client.On("ListImages").Return([]*dockerclient.Image{}, nil)

	// connect client
	engine.ConnectWithClient(client)

	// add engine to cluster
	c.engines[engine.ID] = engine

	// import success
	readCloser := nopCloser{bytes.NewBufferString("ok")}
	client.On("ImportImage", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("*io.PipeReader")).Return(readCloser, nil).Once()

	callback := func(what, status string) {
		// import success
		assert.Equal(t, status, "Import success")
	}
	c.Import("-", "testImageOK", "latest", bytes.NewReader(nil), callback)

	// import error
	readCloser = nopCloser{bytes.NewBufferString("error")}
	err := fmt.Errorf("Import error")
	client.On("ImportImage", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("*io.PipeReader")).Return(readCloser, err).Once()

	callback = func(what, status string) {
		// import error
		assert.Equal(t, status, "Import error")
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
	engine := cluster.NewEngine(id, 0)
	engine.Name = id
	engine.ID = id

	// create mock client
	client := mockclient.NewMockClient()
	client.On("Info").Return(mockInfo, nil)
	client.On("Version").Return(mockVersion, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{}, nil).Once()
	client.On("ListImages").Return([]*dockerclient.Image{}, nil)

	// connect client
	engine.ConnectWithClient(client)

	// add engine to cluster
	c.engines[engine.ID] = engine

	// load success
	client.On("LoadImage", mock.AnythingOfType("*io.PipeReader")).Return(nil).Once()
	callback := func(what, status string) {
		//if load OK, will not come here
		t.Fatalf("Load error")
	}
	c.Load(bytes.NewReader(nil), callback)

	// load error
	err := fmt.Errorf("Load error")
	client.On("LoadImage", mock.AnythingOfType("*io.PipeReader")).Return(err).Once()
	callback = func(what, status string) {
		// load error
		assert.Equal(t, status, "Load error")
	}
	c.Load(bytes.NewReader(nil), callback)
}
