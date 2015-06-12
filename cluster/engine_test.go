package cluster

import (
	"errors"
	"fmt"
	"math"
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

	mockVersion = &dockerclient.Version{
		Version: "1.6.2",
	}
)

func TestEngineConnectionFailure(t *testing.T) {
	engine := NewEngine("test", 0)
	assert.False(t, engine.isConnected())

	// Always fail.
	client := mockclient.NewMockClient()
	client.On("Info").Return(&dockerclient.Info{}, errors.New("fail"))

	// Connect() should fail and isConnected() return false.
	assert.Error(t, engine.ConnectWithClient(client))
	assert.False(t, engine.isConnected())

	client.Mock.AssertExpectations(t)
}

func TestOutdatedEngine(t *testing.T) {
	engine := NewEngine("test", 0)
	client := mockclient.NewMockClient()
	client.On("Info").Return(&dockerclient.Info{}, nil)

	assert.Error(t, engine.ConnectWithClient(client))
	assert.False(t, engine.isConnected())

	client.Mock.AssertExpectations(t)
}

func TestEngineCpusMemory(t *testing.T) {
	engine := NewEngine("test", 0)
	assert.False(t, engine.isConnected())

	client := mockclient.NewMockClient()
	client.On("Info").Return(mockInfo, nil)
	client.On("Version").Return(mockVersion, nil)
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{}, nil)
	client.On("ListImages").Return([]*dockerclient.Image{}, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()

	assert.NoError(t, engine.ConnectWithClient(client))
	assert.True(t, engine.isConnected())
	assert.True(t, engine.IsHealthy())

	assert.Equal(t, engine.UsedCpus(), int64(0))
	assert.Equal(t, engine.UsedMemory(), int64(0))

	client.Mock.AssertExpectations(t)
}

func TestEngineSpecs(t *testing.T) {
	engine := NewEngine("test", 0)
	assert.False(t, engine.isConnected())

	client := mockclient.NewMockClient()
	client.On("Info").Return(mockInfo, nil)
	client.On("Version").Return(mockVersion, nil)
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{}, nil)
	client.On("ListImages").Return([]*dockerclient.Image{}, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()

	assert.NoError(t, engine.ConnectWithClient(client))
	assert.True(t, engine.isConnected())
	assert.True(t, engine.IsHealthy())

	assert.Equal(t, engine.Cpus, mockInfo.NCPU)
	assert.Equal(t, engine.Memory, mockInfo.MemTotal)
	assert.Equal(t, engine.Labels["storagedriver"], mockInfo.Driver)
	assert.Equal(t, engine.Labels["executiondriver"], mockInfo.ExecutionDriver)
	assert.Equal(t, engine.Labels["kernelversion"], mockInfo.KernelVersion)
	assert.Equal(t, engine.Labels["operatingsystem"], mockInfo.OperatingSystem)
	assert.Equal(t, engine.Labels["foo"], "bar")

	client.Mock.AssertExpectations(t)
}

func TestEngineState(t *testing.T) {
	engine := NewEngine("test", 0)
	assert.False(t, engine.isConnected())

	client := mockclient.NewMockClient()
	client.On("Info").Return(mockInfo, nil)
	client.On("Version").Return(mockVersion, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()

	// The client will return one container at first, then a second one will appear.
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{{Id: "one"}}, nil).Once()
	client.On("ListImages").Return([]*dockerclient.Image{}, nil).Once()
	client.On("InspectContainer", "one").Return(&dockerclient.ContainerInfo{Config: &dockerclient.ContainerConfig{CpuShares: 100}}, nil).Once()
	client.On("ListContainers", true, false, fmt.Sprintf("{%q:[%q]}", "id", "two")).Return([]dockerclient.Container{{Id: "two"}}, nil).Once()
	client.On("InspectContainer", "two").Return(&dockerclient.ContainerInfo{Config: &dockerclient.ContainerConfig{CpuShares: 100}}, nil).Once()

	assert.NoError(t, engine.ConnectWithClient(client))
	assert.True(t, engine.isConnected())

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

func TestCreateContainer(t *testing.T) {
	var (
		config = &ContainerConfig{dockerclient.ContainerConfig{
			Image:     "busybox",
			CpuShares: 1,
			Cmd:       []string{"date"},
			Tty:       false,
		}}
		engine = NewEngine("test", 0)
		client = mockclient.NewMockClient()
	)

	client.On("Info").Return(mockInfo, nil)
	client.On("Version").Return(mockVersion, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{}, nil).Once()
	client.On("ListImages").Return([]*dockerclient.Image{}, nil).Once()
	assert.NoError(t, engine.ConnectWithClient(client))
	assert.True(t, engine.isConnected())

	mockConfig := config.ContainerConfig
	mockConfig.CpuShares = int64(math.Ceil(float64(config.CpuShares*1024) / float64(mockInfo.NCPU)))
	mockConfig.HostConfig.CpuShares = mockConfig.CpuShares

	// Everything is ok
	name := "test1"
	id := "id1"
	client.On("CreateContainer", &mockConfig, name).Return(id, nil).Once()
	client.On("ListContainers", true, false, fmt.Sprintf(`{"id":[%q]}`, id)).Return([]dockerclient.Container{{Id: id}}, nil).Once()
	client.On("ListImages").Return([]*dockerclient.Image{}, nil).Once()
	client.On("InspectContainer", id).Return(&dockerclient.ContainerInfo{Config: &config.ContainerConfig}, nil).Once()
	container, err := engine.Create(config, name, false)
	assert.Nil(t, err)
	assert.Equal(t, container.Id, id)
	assert.Len(t, engine.Containers(), 1)

	// Image not found, pullImage == false
	name = "test2"
	mockConfig.CpuShares = int64(math.Ceil(float64(config.CpuShares*1024) / float64(mockInfo.NCPU)))
	client.On("CreateContainer", &mockConfig, name).Return("", dockerclient.ErrNotFound).Once()
	container, err = engine.Create(config, name, false)
	assert.Equal(t, err, dockerclient.ErrNotFound)
	assert.Nil(t, container)

	// Image not found, pullImage == true, and the image can be pulled successfully
	name = "test3"
	id = "id3"
	mockConfig.CpuShares = int64(math.Ceil(float64(config.CpuShares*1024) / float64(mockInfo.NCPU)))
	client.On("PullImage", config.Image+":latest", mock.Anything).Return(nil).Once()
	client.On("CreateContainer", &mockConfig, name).Return("", dockerclient.ErrNotFound).Once()
	client.On("CreateContainer", &mockConfig, name).Return(id, nil).Once()
	client.On("ListContainers", true, false, fmt.Sprintf(`{"id":[%q]}`, id)).Return([]dockerclient.Container{{Id: id}}, nil).Once()
	client.On("ListImages").Return([]*dockerclient.Image{}, nil).Once()
	client.On("InspectContainer", id).Return(&dockerclient.ContainerInfo{Config: &config.ContainerConfig}, nil).Once()
	container, err = engine.Create(config, name, true)
	assert.Nil(t, err)
	assert.Equal(t, container.Id, id)
	assert.Len(t, engine.Containers(), 2)
}

func TestTotalMemory(t *testing.T) {
	engine := NewEngine("test", 0.05)
	engine.Memory = 1024
	assert.Equal(t, engine.TotalMemory(), int64(1024+1024*5/100))

	engine = NewEngine("test", 0)
	engine.Memory = 1024
	assert.Equal(t, engine.TotalMemory(), int64(1024))
}

func TestTotalCpus(t *testing.T) {
	engine := NewEngine("test", 0.05)
	engine.Cpus = 2
	assert.Equal(t, engine.TotalCpus(), int64(2+2*5/100))

	engine = NewEngine("test", 0)
	engine.Cpus = 2
	assert.Equal(t, engine.TotalCpus(), int64(2))
}

func TestUsedCpus(t *testing.T) {
	var (
		containerNcpu = []int64{1, 2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47}
		hostNcpu      = []int64{1, 2, 4, 8, 10, 12, 16, 20, 32, 36, 40, 48}
	)

	engine := NewEngine("test", 0)
	client := mockclient.NewMockClient()

	for _, hn := range hostNcpu {
		for _, cn := range containerNcpu {
			if cn <= hn {
				mockInfo.NCPU = hn
				cpuShares := int64(math.Ceil(float64(cn*1024) / float64(mockInfo.NCPU)))

				client.On("Info").Return(mockInfo, nil)
				client.On("Version").Return(mockVersion, nil)
				client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()
				client.On("ListImages").Return([]*dockerclient.Image{}, nil).Once()
				client.On("ListContainers", true, false, "").Return([]dockerclient.Container{{Id: "test"}}, nil).Once()
				client.On("InspectContainer", "test").Return(&dockerclient.ContainerInfo{Config: &dockerclient.ContainerConfig{CpuShares: cpuShares}}, nil).Once()
				engine.ConnectWithClient(client)

				assert.Equal(t, engine.UsedCpus(), cn)
			}
		}
	}
}
