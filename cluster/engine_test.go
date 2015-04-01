package cluster

import (
	"errors"
	"fmt"
	"testing"

	"github.com/samalba/dockerclient"
	"github.com/samalba/dockerclient/mockclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
)

func TestEngineConnectionFailure(t *testing.T) {
	engine := NewEngine("test", "", 0)
	assert.False(t, engine.IsConnected())

	// Always fail.
	client := mockclient.NewMockClient()
	client.On("Info").Return(&dockerclient.Info{}, errors.New("fail"))

	// Connect() should fail and IsConnected() return false.
	assert.Error(t, engine.ConnectClient(client))
	assert.False(t, engine.IsConnected())

	client.Mock.AssertExpectations(t)
}

func TestOutdatedEngine(t *testing.T) {
	engine := NewEngine("test", "", 0)
	client := mockclient.NewMockClient()
	client.On("Info").Return(&dockerclient.Info{}, nil)

	assert.Error(t, engine.ConnectClient(client))
	assert.False(t, engine.IsConnected())

	client.Mock.AssertExpectations(t)
}

func TestEngineCpusMemory(t *testing.T) {
	engine := NewEngine("test", "", 0)
	assert.False(t, engine.IsConnected())

	client := mockclient.NewMockClient()
	client.On("Info").Return(mockInfo, nil)
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{}, nil)
	client.On("ListImages").Return([]*dockerclient.Image{}, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()

	assert.NoError(t, engine.ConnectClient(client))
	assert.True(t, engine.IsConnected())
	assert.True(t, engine.IsHealthy())

	assert.Equal(t, engine.UsedCpus(), 0)
	assert.Equal(t, engine.UsedMemory(), 0)

	client.Mock.AssertExpectations(t)
}

func TestEngineSpecs(t *testing.T) {
	engine := NewEngine("test", "", 0)
	assert.False(t, engine.IsConnected())

	client := mockclient.NewMockClient()
	client.On("Info").Return(mockInfo, nil)
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{}, nil)
	client.On("ListImages").Return([]*dockerclient.Image{}, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()

	assert.NoError(t, engine.ConnectClient(client))
	assert.True(t, engine.IsConnected())
	assert.True(t, engine.IsHealthy())

	assert.Equal(t, engine.Cpus, mockInfo.NCPU)
	assert.Equal(t, engine.Memory, mockInfo.MemTotal)
	assert.Equal(t, engine.Labels()["storagedriver"], mockInfo.Driver)
	assert.Equal(t, engine.Labels()["executiondriver"], mockInfo.ExecutionDriver)
	assert.Equal(t, engine.Labels()["kernelversion"], mockInfo.KernelVersion)
	assert.Equal(t, engine.Labels()["operatingsystem"], mockInfo.OperatingSystem)
	assert.Equal(t, engine.Labels()["foo"], "bar")

	client.Mock.AssertExpectations(t)
}

func TestEngineState(t *testing.T) {
	engine := NewEngine("test", "", 0)
	assert.False(t, engine.IsConnected())

	client := mockclient.NewMockClient()
	client.On("Info").Return(mockInfo, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()

	// The client will return one container at first, then a second one will appear.
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{{Id: "one"}}, nil).Once()
	client.On("ListImages").Return([]*dockerclient.Image{}, nil).Once()
	client.On("InspectContainer", "one").Return(&dockerclient.ContainerInfo{Config: &dockerclient.ContainerConfig{CpuShares: 100}}, nil).Once()
	client.On("ListContainers", true, false, fmt.Sprintf("{%q:[%q]}", "id", "two")).Return([]dockerclient.Container{{Id: "two"}}, nil).Once()
	client.On("InspectContainer", "two").Return(&dockerclient.ContainerInfo{Config: &dockerclient.ContainerConfig{CpuShares: 100}}, nil).Once()

	assert.NoError(t, engine.ConnectClient(client))
	assert.True(t, engine.IsConnected())

	// The engine should only have a single container at this point.
	containers := engine.Containers()
	assert.Len(t, containers, 1)
	if containers[0].Id != "one" {
		t.Fatalf("Missing container: one")
	}

	// Fake an event which will trigger a refresh. The second container will appear.
	engine.handler(&dockerclient.Event{Id: "two", Status: "created"}, nil)
	containers = engine.Containers()
	assert.Len(t, containers, 2)
	if containers[0].Id != "one" && containers[1].Id != "one" {
		t.Fatalf("Missing container: one")
	}
	if containers[0].Id != "two" && containers[1].Id != "two" {
		t.Fatalf("Missing container: two")
	}

	client.Mock.AssertExpectations(t)
}

func TestEngineContainerLookup(t *testing.T) {
	engine := NewEngine("test-engine", "", 0)
	assert.False(t, engine.IsConnected())

	client := mockclient.NewMockClient()
	client.On("Info").Return(mockInfo, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()

	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{{Id: "container-id", Names: []string{"/container-name1", "/container-name2"}}}, nil).Once()
	client.On("ListImages").Return([]*dockerclient.Image{}, nil).Once()
	client.On("InspectContainer", "container-id").Return(&dockerclient.ContainerInfo{Config: &dockerclient.ContainerConfig{CpuShares: 100}}, nil).Once()

	assert.NoError(t, engine.ConnectClient(client))
	assert.True(t, engine.IsConnected())

	// Invalid lookup
	assert.Nil(t, engine.Container("invalid-id"))
	assert.Nil(t, engine.Container(""))
	// Container ID lookup.
	assert.NotNil(t, engine.Container("container-id"))
	// Container ID prefix lookup.
	assert.NotNil(t, engine.Container("container-"))
	// Container name lookup.
	assert.NotNil(t, engine.Container("container-name1"))
	assert.NotNil(t, engine.Container("container-name2"))
	// Container engine/name matching.
	assert.NotNil(t, engine.Container("id/container-name1"))
	assert.NotNil(t, engine.Container("id/container-name2"))

	client.Mock.AssertExpectations(t)
}

func TestTotalMemory(t *testing.T) {
	engine := NewEngine("test", "", 0.05)
	engine.Memory = 1024
	assert.Equal(t, engine.TotalMemory(), 1024+1024*5/100)

	engine = NewEngine("test", "", 0)
	engine.Memory = 1024
	assert.Equal(t, engine.TotalMemory(), 1024)
}

func TestTotalCpus(t *testing.T) {
	engine := NewEngine("test", "", 0.05)
	engine.Cpus = 2
	assert.Equal(t, engine.TotalCpus(), 2+2*5/100)

	engine = NewEngine("test", "", 0)
	engine.Cpus = 2
	assert.Equal(t, engine.TotalCpus(), 2)
}
