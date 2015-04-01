package swarm

import (
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

func TestCreateContainer(t *testing.T) {
	var (
		config = &dockerclient.ContainerConfig{
			Image:     "busybox",
			CpuShares: 512,
			Cmd:       []string{"date"},
			Tty:       false,
		}
		node   = NewNode("test", 0)
		client = mockclient.NewMockClient()
	)

	client.On("Info").Return(mockInfo, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything, mock.Anything).Return()
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{}, nil).Once()
	client.On("ListImages").Return([]*dockerclient.Image{}, nil).Once()
	assert.NoError(t, node.ConnectClient(client))
	assert.True(t, node.IsConnected())

	mockConfig := *config
	mockConfig.CpuShares = config.CpuShares * mockInfo.NCPU

	// Everything is ok
	name := "test1"
	id := "id1"
	client.On("CreateContainer", &mockConfig, name).Return(id, nil).Once()
	client.On("ListContainers", true, false, fmt.Sprintf(`{"id":[%q]}`, id)).Return([]dockerclient.Container{{Id: id}}, nil).Once()
	client.On("ListImages").Return([]*dockerclient.Image{}, nil).Once()
	client.On("InspectContainer", id).Return(&dockerclient.ContainerInfo{Config: config}, nil).Once()
	container, err := node.create(config, name, false)
	assert.Nil(t, err)
	assert.Equal(t, container.Id, id)
	assert.Len(t, node.Containers(), 1)

	// Image not found, pullImage == false
	name = "test2"
	mockConfig.CpuShares = config.CpuShares * mockInfo.NCPU
	client.On("CreateContainer", &mockConfig, name).Return("", dockerclient.ErrNotFound).Once()
	container, err = node.create(config, name, false)
	assert.Equal(t, err, dockerclient.ErrNotFound)
	assert.Nil(t, container)

	// Image not found, pullImage == true, and the image can be pulled successfully
	name = "test3"
	id = "id3"
	mockConfig.CpuShares = config.CpuShares * mockInfo.NCPU
	client.On("PullImage", config.Image+":latest", mock.Anything).Return(nil).Once()
	client.On("CreateContainer", &mockConfig, name).Return("", dockerclient.ErrNotFound).Once()
	client.On("CreateContainer", &mockConfig, name).Return(id, nil).Once()
	client.On("ListContainers", true, false, fmt.Sprintf(`{"id":[%q]}`, id)).Return([]dockerclient.Container{{Id: id}}, nil).Once()
	client.On("ListImages").Return([]*dockerclient.Image{}, nil).Once()
	client.On("InspectContainer", id).Return(&dockerclient.ContainerInfo{Config: config}, nil).Once()
	container, err = node.create(config, name, true)
	assert.Nil(t, err)
	assert.Equal(t, container.Id, id)
	assert.Len(t, node.Containers(), 2)
}
