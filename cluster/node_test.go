package cluster

import (
	"errors"
	"fmt"
	"testing"

	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	mockInfo = &dockerclient.Info{
		NCPU:            10,
		MemTotal:        20,
		Driver:          "driver-test",
		ExecutionDriver: "execution-driver-test",
		KernelVersion:   "1.2.3",
		OperatingSystem: "golang",
	}
)

func TestNodeConnectionFailure(t *testing.T) {
	node := NewNode("test", "test")
	assert.False(t, node.IsConnected())

	// Always fail.
	client := dockerclient.NewMockClient()
	client.On("Info").Return(&dockerclient.Info{}, errors.New("fail"))

	// Connect() should fail and IsConnected() return false.
	assert.Error(t, node.connectClient(client))
	assert.False(t, node.IsConnected())

	client.Mock.AssertExpectations(t)
}

func TestNodeSpecs(t *testing.T) {
	node := NewNode("test", "test")
	assert.False(t, node.IsConnected())

	client := dockerclient.NewMockClient()
	client.On("Info").Return(mockInfo, nil)
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{}, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything).Return()

	assert.NoError(t, node.connectClient(client))
	assert.True(t, node.IsConnected())

	assert.Equal(t, node.Cpus, mockInfo.NCPU)
	assert.Equal(t, node.Memory, mockInfo.MemTotal)
	assert.Equal(t, node.Labels["storagedriver"], mockInfo.Driver)
	assert.Equal(t, node.Labels["executiondriver"], mockInfo.ExecutionDriver)
	assert.Equal(t, node.Labels["kernelversion"], mockInfo.KernelVersion)
	assert.Equal(t, node.Labels["operatingsystem"], mockInfo.OperatingSystem)

	client.Mock.AssertExpectations(t)
}

func TestNodeState(t *testing.T) {
	node := NewNode("test", "test")
	assert.False(t, node.IsConnected())

	client := dockerclient.NewMockClient()
	client.On("Info").Return(mockInfo, nil)
	client.On("StartMonitorEvents", mock.Anything, mock.Anything).Return()

	// The client will return one container at first, then a second one will appear.
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{{Id: "one"}}, nil).Once()
	client.On("InspectContainer", "one").Return(&dockerclient.ContainerInfo{}, nil).Once()
	client.On("ListContainers", true, false, fmt.Sprintf("{%q:[%q]}", "id", "two")).Return([]dockerclient.Container{{Id: "two"}}, nil).Once()
	client.On("InspectContainer", "two").Return(&dockerclient.ContainerInfo{}, nil).Once()

	assert.NoError(t, node.connectClient(client))
	assert.True(t, node.IsConnected())

	// The node should only have a single container at this point.
	assert.Len(t, node.Containers(), 1)
	if _, ok := node.Containers()["one"]; !ok {
		t.Fatalf("Missing container: one")
	}

	// Fake an event which will trigger a refresh. The second container will appear.
	node.handler(&dockerclient.Event{Id: "two", Status: "created"})
	assert.Len(t, node.Containers(), 2)
	if _, ok := node.Containers()["one"]; !ok {
		t.Fatalf("Missing container: one")
	}
	if _, ok := node.Containers()["two"]; !ok {
		t.Fatalf("Missing container: two")
	}

	client.Mock.AssertExpectations(t)
}
