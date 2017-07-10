package cluster

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	engineapi "github.com/docker/docker/client"
	engineapimock "github.com/docker/swarm/api/mockclient"
	engineapinop "github.com/docker/swarm/api/nopclient"
	"github.com/samalba/dockerclient"
	"github.com/samalba/dockerclient/mockclient"
	"github.com/samalba/dockerclient/nopclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	mockInfo = types.Info{
		ID:              "id",
		Name:            "name",
		NCPU:            10,
		MemTotal:        20,
		Driver:          "driver-test",
		KernelVersion:   "1.2.3",
		OperatingSystem: "golang",
		OSType:          "linux",
		Labels:          []string{"foo=bar"},
	}

	mockVersion = types.Version{
		Version: "1.8.2",
	}

	engOpts = &EngineOpts{
		RefreshMinInterval: time.Duration(30) * time.Second,
		RefreshMaxInterval: time.Duration(60) * time.Second,
		FailureRetry:       3,
	}
)

type nopCloser struct {
	io.Reader
}

// Close
func (nopCloser) Close() error {
	return nil
}

func TestSetEngineState(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	assert.True(t, engine.state == statePending)
	engine.setState(stateUnhealthy)
	assert.True(t, engine.state == stateUnhealthy)
	engine.setState(stateHealthy)
	assert.True(t, engine.state == stateHealthy)
}

func TestErrMsg(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	assert.True(t, len(engine.ErrMsg()) == 0)
	message := "cannot connect"
	engine.setErrMsg(message)
	assert.True(t, engine.ErrMsg() == message)
}

func TestCheckConnectionErr(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	engine.setState(stateHealthy)
	assert.True(t, engine.failureCount == 0)
	err := dockerclient.ErrConnectionRefused
	engine.CheckConnectionErr(err)
	assert.True(t, len(engine.ErrMsg()) > 0)
	assert.True(t, engine.failureCount == 1)

	err = engineapi.ErrorConnectionFailed("")
	engine.CheckConnectionErr(err)
	assert.True(t, engine.failureCount == 2)

	err = nil
	engine.CheckConnectionErr(err)
	assert.True(t, engine.failureCount == 0)
	assert.True(t, len(engine.ErrMsg()) == 0)
	// Do not accept random error
	err = fmt.Errorf("random error")
	engine.CheckConnectionErr(err)
	assert.True(t, engine.failureCount == 0)
	assert.True(t, len(engine.ErrMsg()) == 0)
}

func TestEngineFailureCount(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	engine.setState(stateHealthy)
	for i := 0; i < engine.opts.FailureRetry; i++ {
		assert.True(t, engine.IsHealthy())
		engine.incFailureCount()
	}
	assert.False(t, engine.IsHealthy())
	assert.True(t, engine.failureCount == engine.opts.FailureRetry)
	engine.resetFailureCount()
	assert.True(t, engine.failureCount == 0)
}

func TestHealthIndicator(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	assert.True(t, engine.state == statePending)
	assert.True(t, engine.HealthIndicator() == 0)
	engine.setState(stateUnhealthy)
	assert.True(t, engine.HealthIndicator() == 0)
	engine.setState(stateHealthy)
	assert.True(t, engine.HealthIndicator() == 100)
	engine.incFailureCount()
	assert.True(t, engine.HealthIndicator() == (int64)(100-100/engine.opts.FailureRetry))
}

func TestEngineConnectionFailure(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	assert.False(t, engine.isConnected())

	// Always fail.
	client := mockclient.NewMockClient()
	apiClient := engineapimock.NewMockClient()
	apiClient.On("Info", mock.Anything).Return(types.Info{}, errors.New("fail"))

	// Connect() should fail
	assert.Error(t, engine.ConnectWithClient(client, apiClient))

	// isConnected() should return false
	nop := nopclient.NewNopClient()
	nopAPIClient := engineapinop.NewNopClient()
	assert.Error(t, engine.ConnectWithClient(nop, nopAPIClient))
	assert.False(t, engine.isConnected())

	client.Mock.AssertExpectations(t)
	apiClient.Mock.AssertExpectations(t)
}

func TestOutdatedEngine(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	client := mockclient.NewMockClient()
	apiClient := engineapimock.NewMockClient()
	apiClient.On("Info", mock.Anything).Return(types.Info{}, nil)

	assert.Error(t, engine.ConnectWithClient(client, apiClient))

	nop := nopclient.NewNopClient()
	nopAPIClient := engineapinop.NewNopClient()
	assert.Error(t, engine.ConnectWithClient(nop, nopAPIClient))
	assert.False(t, engine.isConnected())

	client.Mock.AssertExpectations(t)
	apiClient.Mock.AssertExpectations(t)
}

func TestEngineCpusMemory(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	engine.setState(stateUnhealthy)
	assert.False(t, engine.isConnected())

	client := mockclient.NewMockClient()
	apiClient := engineapimock.NewMockClient()
	apiClient.On("Info", mock.Anything).Return(mockInfo, nil)
	apiClient.On("ServerVersion", mock.Anything).Return(mockVersion, nil)
	apiClient.On("NetworkList", mock.Anything,
		mock.AnythingOfType("NetworkListOptions"),
	).Return([]types.NetworkResource{}, nil)
	apiClient.On("VolumeList", mock.Anything,
		mock.AnythingOfType("Args"),
	).Return(volume.VolumesListOKBody{}, nil)
	apiClient.On("ImageList", mock.Anything, mock.AnythingOfType("ImageListOptions")).Return([]types.ImageSummary{}, nil)
	apiClient.On("ContainerList", mock.Anything, types.ContainerListOptions{All: true, Size: false}).Return([]types.Container{}, nil)
	apiClient.On("Events", mock.Anything, mock.AnythingOfType("EventsOptions")).Return(make(chan events.Message), make(chan error))
	apiClient.On("NegotiateAPIVersion", mock.Anything).Return()

	assert.NoError(t, engine.ConnectWithClient(client, apiClient))
	assert.True(t, engine.isConnected())
	assert.True(t, engine.IsHealthy())

	assert.Equal(t, engine.UsedCpus(), int64(0))
	assert.Equal(t, engine.UsedMemory(), int64(0))

	client.Mock.AssertExpectations(t)
	apiClient.Mock.AssertExpectations(t)
}

func TestEngineSpecs(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	engine.setState(stateUnhealthy)
	assert.False(t, engine.isConnected())

	client := mockclient.NewMockClient()
	apiClient := engineapimock.NewMockClient()
	apiClient.On("Info", mock.Anything).Return(mockInfo, nil)
	apiClient.On("ServerVersion", mock.Anything).Return(mockVersion, nil)
	apiClient.On("NetworkList", mock.Anything,
		mock.AnythingOfType("NetworkListOptions"),
	).Return([]types.NetworkResource{}, nil)
	apiClient.On("VolumeList", mock.Anything,
		mock.AnythingOfType("Args"),
	).Return(volume.VolumesListOKBody{}, nil)
	apiClient.On("ImageList", mock.Anything, mock.AnythingOfType("ImageListOptions")).Return([]types.ImageSummary{}, nil)
	apiClient.On("ContainerList", mock.Anything, types.ContainerListOptions{All: true, Size: false}).Return([]types.Container{}, nil)
	apiClient.On("Events", mock.Anything, mock.AnythingOfType("EventsOptions")).Return(make(chan events.Message), make(chan error))
	apiClient.On("NegotiateAPIVersion", mock.Anything).Return()

	assert.NoError(t, engine.ConnectWithClient(client, apiClient))
	assert.True(t, engine.isConnected())
	assert.True(t, engine.IsHealthy())

	mockInfo2 := mockInfo
	mockInfo2.Labels = []string{"foo=bar", "executiondriver=newdriver", "node=node1"}

	assert.Equal(t, engine.Cpus, int64(mockInfo2.NCPU))
	assert.Equal(t, engine.Memory, mockInfo2.MemTotal)
	assert.Equal(t, engine.Labels["storagedriver"], mockInfo2.Driver)

	assert.Equal(t, engine.Labels["kernelversion"], mockInfo2.KernelVersion)
	assert.Equal(t, engine.Labels["operatingsystem"], mockInfo2.OperatingSystem)
	assert.Equal(t, engine.Labels["ostype"], mockInfo2.OSType)
	assert.Equal(t, engine.Labels["foo"], "bar")

	assert.NotEqual(t, engine.Labels["node"], "node1")

	client.Mock.AssertExpectations(t)
	apiClient.Mock.AssertExpectations(t)
}

func TestEngineState(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	engine.setState(stateUnhealthy)
	assert.False(t, engine.isConnected())

	client := mockclient.NewMockClient()
	apiClient := engineapimock.NewMockClient()
	apiClient.On("Info", mock.Anything).Return(mockInfo, nil)
	apiClient.On("ServerVersion", mock.Anything).Return(mockVersion, nil)
	apiClient.On("NetworkList", mock.Anything,
		mock.AnythingOfType("NetworkListOptions"),
	).Return([]types.NetworkResource{}, nil)
	apiClient.On("VolumeList", mock.Anything,
		mock.AnythingOfType("Args"),
	).Return(volume.VolumesListOKBody{}, nil)
	apiClient.On("Events", mock.Anything, mock.AnythingOfType("EventsOptions")).Return(make(chan events.Message), make(chan error))
	apiClient.On("NegotiateAPIVersion", mock.Anything).Return()

	// The client will return one container at first, then a second one will appear.
	apiClient.On("ImageList", mock.Anything, mock.AnythingOfType("ImageListOptions")).Return([]types.ImageSummary{}, nil).Once()

	apiClient.On(
		"ContainerList",
		mock.Anything,
		types.ContainerListOptions{
			All:  true,
			Size: false,
		},
	).Return(
		[]types.Container{
			{
				ID: "one",
			},
		},
		nil,
	).Once()

	apiClient.On("ContainerInspect", mock.Anything, "one").Return(
		types.ContainerJSON{
			ContainerJSONBase: &types.ContainerJSONBase{
				HostConfig: &containertypes.HostConfig{
					Resources: containertypes.Resources{
						CPUShares: 100,
					},
				},
				State: &types.ContainerState{
					StartedAt:  "2016-06-06T01:41:38.090313266Z",
					FinishedAt: "0001-01-01T00:00:00Z",
				},
			},
			Config: &containertypes.Config{},
			NetworkSettings: &types.NetworkSettings{
				Networks: nil,
			},
		},
		nil,
	).Once()

	filterArgs := filters.NewArgs()
	filterArgs.Add("id", "two")

	apiClient.On(
		"ContainerList",
		mock.Anything,
		types.ContainerListOptions{
			All:     true,
			Size:    false,
			Filters: filterArgs,
		},
	).Return(
		[]types.Container{{ID: "two"}},
		nil,
	).Once()

	apiClient.On("ContainerInspect", mock.Anything, "two").Return(
		types.ContainerJSON{
			ContainerJSONBase: &types.ContainerJSONBase{
				HostConfig: &containertypes.HostConfig{
					Resources: containertypes.Resources{
						CPUShares: 100,
					},
				},
				State: &types.ContainerState{
					StartedAt:  "2016-06-06T01:41:38.090313266Z",
					FinishedAt: "0001-01-01T00:00:00Z",
				},
			},
			Config: &containertypes.Config{},
			NetworkSettings: &types.NetworkSettings{
				Networks: nil,
			},
		},
		nil,
	).Once()

	assert.NoError(t, engine.ConnectWithClient(client, apiClient))
	assert.True(t, engine.isConnected())

	// The engine should only have a single container at this point.
	containers := engine.Containers()
	assert.Len(t, containers, 1)
	if containers[0].ID != "one" {
		t.Fatalf("Missing container: one")
	}

	// Fake an event which will trigger a refresh. The second container will appear.
	engine.handler(events.Message{ID: "two", Status: "created"})
	containers = engine.Containers()
	assert.Len(t, containers, 2)
	if containers[0].ID != "one" && containers[1].ID != "one" {
		t.Fatalf("Missing container: one")
	}
	if containers[0].ID != "two" && containers[1].ID != "two" {
		t.Fatalf("Missing container: two")
	}

	client.Mock.AssertExpectations(t)
	apiClient.Mock.AssertExpectations(t)
}

func TestCreateContainer(t *testing.T) {
	var (
		config = &ContainerConfig{containertypes.Config{
			Image: "busybox",
			Cmd:   []string{"date"},
			Tty:   false,
		}, containertypes.HostConfig{
			Resources: containertypes.Resources{
				CPUShares: 1,
			},
		}, networktypes.NetworkingConfig{}}
		state = types.ContainerState{
			StartedAt:  "2016-06-06T01:41:38.090313266Z",
			FinishedAt: "0001-01-01T00:00:00Z",
		}
		engine     = NewEngine("test", 0, engOpts)
		client     = mockclient.NewMockClient()
		apiClient  = engineapimock.NewMockClient()
		readCloser = nopCloser{bytes.NewBufferString("")}
	)

	engine.setState(stateUnhealthy)
	apiClient.On("Info", mock.Anything).Return(mockInfo, nil)
	apiClient.On("ServerVersion", mock.Anything).Return(mockVersion, nil)
	apiClient.On("NetworkList", mock.Anything,
		mock.AnythingOfType("NetworkListOptions"),
	).Return([]types.NetworkResource{}, nil)
	apiClient.On("VolumeList", mock.Anything,
		mock.AnythingOfType("Args"),
	).Return(volume.VolumesListOKBody{}, nil)
	apiClient.On("Events", mock.Anything, mock.AnythingOfType("EventsOptions")).Return(make(chan events.Message), make(chan error))
	client.On("ListContainers", true, false, "").Return([]dockerclient.Container{}, nil).Once()
	apiClient.On("ImageList", mock.Anything, mock.AnythingOfType("ImageListOptions")).Return([]types.ImageSummary{}, nil).Once()
	// filterArgs1 := filters.NewArgs()
	// filterArgs1.Add("id", id)
	apiClient.On("ContainerList", mock.Anything, types.ContainerListOptions{All: true, Size: false}).Return([]types.Container{}, nil).Once()
	apiClient.On("NegotiateAPIVersion", mock.Anything).Return()

	assert.NoError(t, engine.ConnectWithClient(client, apiClient))
	assert.True(t, engine.isConnected())

	mockConfig := *config
	mockConfig.HostConfig.CPUShares = int64(math.Ceil(float64(config.HostConfig.CPUShares*1024) / float64(mockInfo.NCPU)))

	// Everything is ok
	name := "test1"
	id := "id1"
	apiClient.On("ImageList", mock.Anything, mock.AnythingOfType("ImageListOptions")).Return([]types.ImageSummary{}, nil).Once()
	var auth *types.AuthConfig

	apiClient.On(
		"ContainerCreate",
		mock.Anything,
		&mockConfig.Config,
		&mockConfig.HostConfig,
		&mockConfig.NetworkingConfig,
		name,
	).Return(
		containertypes.ContainerCreateCreatedBody{ID: id},
		nil,
	).Once()

	filterArgs := filters.NewArgs()
	filterArgs.Add("id", id)

	apiClient.On(
		"ContainerList",
		mock.Anything,
		types.ContainerListOptions{
			All:     true,
			Size:    false,
			Filters: filterArgs,
		},
	).Return(
		[]types.Container{{ID: id}},
		nil,
	).Once()

	apiClient.On(
		"ContainerInspect",
		mock.Anything,
		id,
	).Return(
		types.ContainerJSON{
			Config: &config.Config,
			ContainerJSONBase: &types.ContainerJSONBase{
				HostConfig: &config.HostConfig,
				State:      &state,
			},
			NetworkSettings: &types.NetworkSettings{
				Networks: nil,
			},
		},
		nil,
	).Once()

	container, err := engine.CreateContainer(config, name, false, auth)
	assert.Nil(t, err)
	assert.Equal(t, container.ID, id)
	assert.Len(t, engine.Containers(), 1)

	// Image not found, pullImage == false
	name = "test2"
	mockConfig.HostConfig.CPUShares = int64(math.Ceil(float64(config.HostConfig.CPUShares*1024) / float64(mockInfo.NCPU)))
	// FIXMEENGINEAPI : below should return an docker/api error, or something custom
	apiClient.On(
		"ContainerCreate",
		mock.Anything,
		&mockConfig.Config,
		&mockConfig.HostConfig,
		&mockConfig.NetworkingConfig,
		name,
	).Return(
		containertypes.ContainerCreateCreatedBody{},
		dockerclient.ErrImageNotFound,
	).Once()

	container, err = engine.CreateContainer(config, name, false, auth)
	assert.Equal(t, err, dockerclient.ErrImageNotFound)
	assert.Nil(t, container)

	// Image not found, pullImage == true, and the image can be pulled successfully
	name = "test3"
	id = "id3"
	apiClient.On("ImageList", mock.Anything, mock.AnythingOfType("ImageListOptions")).Return([]types.ImageSummary{}, nil).Once()
	mockConfig.HostConfig.CPUShares = int64(math.Ceil(float64(config.HostConfig.CPUShares*1024) / float64(mockInfo.NCPU)))
	apiClient.On("ImagePull", mock.Anything, config.Image, mock.AnythingOfType("types.ImagePullOptions")).Return(readCloser, nil).Once()
	// TODO(nishanttotla): below should return an docker/api error, or something custom, so that we can get rid of dockerclient
	apiClient.On(
		"ContainerCreate",
		mock.Anything,
		&mockConfig.Config,
		&mockConfig.HostConfig,
		&mockConfig.NetworkingConfig,
		name,
	).Return(
		containertypes.ContainerCreateCreatedBody{},
		dockerclient.ErrImageNotFound,
	).Once()

	// FIXMEENGINEAPI : below should return an docker/api error, or something custom
	apiClient.On(
		"ContainerCreate",
		mock.Anything,
		&mockConfig.Config,
		&mockConfig.HostConfig,
		&mockConfig.NetworkingConfig,
		name,
	).Return(
		containertypes.ContainerCreateCreatedBody{ID: id},
		nil,
	).Once()

	filterArgs = filters.NewArgs()
	filterArgs.Add("id", id)
	apiClient.On(
		"ContainerList",
		mock.Anything,
		types.ContainerListOptions{
			All:     true,
			Size:    false,
			Filters: filterArgs,
		},
	).Return(
		[]types.Container{{ID: id}},
		nil,
	).Once()

	apiClient.On("ContainerInspect", mock.Anything, id).Return(
		types.ContainerJSON{
			Config: &config.Config,
			ContainerJSONBase: &types.ContainerJSONBase{
				HostConfig: &config.HostConfig,
				State:      &state,
			},
			NetworkSettings: &types.NetworkSettings{
				Networks: nil,
			},
		},
		nil,
	).Once()

	container, err = engine.CreateContainer(config, name, true, auth)
	assert.Nil(t, err)
	assert.Equal(t, container.ID, id)
	assert.Len(t, engine.Containers(), 2)
}

func TestImages(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)
	engine.setState(stateHealthy)
	engine.images = []*Image{
		{types.ImageSummary{ID: "a"}, engine},
		{types.ImageSummary{ID: "b"}, engine},
		{types.ImageSummary{ID: "c"}, engine},
	}

	result := engine.Images()
	assert.Equal(t, len(result), 3)
}

func TestTotalMemory(t *testing.T) {
	engine := NewEngine("test", 0.05, engOpts)
	engine.Memory = 1024
	assert.Equal(t, engine.TotalMemory(), int64(1024+1024*5/100))

	engine = NewEngine("test", 0, engOpts)
	engine.Memory = 1024
	assert.Equal(t, engine.TotalMemory(), int64(1024))
}

func TestTotalCpus(t *testing.T) {
	engine := NewEngine("test", 0.05, engOpts)
	engine.Cpus = 2
	assert.Equal(t, engine.TotalCpus(), int64(2+2*5/100))

	engine = NewEngine("test", 0, engOpts)
	engine.Cpus = 2
	assert.Equal(t, engine.TotalCpus(), int64(2))
}

func TestUsedCpus(t *testing.T) {
	var (
		containerNcpu = []int64{1, 2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47}
		hostNcpu      = []int64{1, 2, 4, 8, 10, 12, 16, 20, 32, 36, 40, 48}
	)

	engine := NewEngine("test", 0, engOpts)
	engine.setState(stateHealthy)
	client := mockclient.NewMockClient()
	apiClient := engineapimock.NewMockClient()

	for _, hn := range hostNcpu {
		for _, cn := range containerNcpu {
			if cn <= hn {
				mockInfo.NCPU = int(hn)
				cpuShares := int64(math.Ceil(float64(cn*1024) / float64(mockInfo.NCPU)))

				apiClient.On("Info", mock.Anything).Return(mockInfo, nil).Once()
				apiClient.On("ServerVersion", mock.Anything).Return(mockVersion, nil)
				apiClient.On("NetworkList", mock.Anything,
					mock.AnythingOfType("NetworkListOptions"),
				).Return([]types.NetworkResource{}, nil)
				apiClient.On("VolumeList", mock.Anything,
					mock.AnythingOfType("Args"),
				).Return(volume.VolumesListOKBody{}, nil)
				apiClient.On("Events", mock.Anything, mock.AnythingOfType("EventsOptions")).Return(make(chan events.Message), make(chan error))
				apiClient.On("ImageList", mock.Anything, mock.AnythingOfType("ImageListOptions")).Return([]types.ImageSummary{}, nil).Once()
				apiClient.On("NegotiateAPIVersion", mock.Anything).Return()

				apiClient.On(
					"ContainerList",
					mock.Anything,
					types.ContainerListOptions{
						All:  true,
						Size: false,
					},
				).Return(
					[]types.Container{{ID: "test"}},
					nil,
				).Once()

				apiClient.On("ContainerInspect", mock.Anything, "test").Return(
					types.ContainerJSON{
						ContainerJSONBase: &types.ContainerJSONBase{
							HostConfig: &containertypes.HostConfig{
								Resources: containertypes.Resources{
									CPUShares: cpuShares,
								},
							},
							State: &types.ContainerState{
								StartedAt:  "2016-06-06T01:41:38.090313266Z",
								FinishedAt: "0001-01-01T00:00:00Z",
							},
						},
						Config: &containertypes.Config{},
						NetworkSettings: &types.NetworkSettings{
							Networks: nil,
						},
					},
					nil,
				).Once()

				engine.ConnectWithClient(client, apiClient)
				assert.Equal(t, engine.Cpus, int64(mockInfo.NCPU))
				assert.Equal(t, engine.UsedCpus(), cn)
			}
		}
	}
}

func TestContainerRemovedDuringRefresh(t *testing.T) {
	var (
		container1 = types.Container{ID: "c1"}
		container2 = types.Container{ID: "c2"}
		info1      types.ContainerJSON
		info2      = types.ContainerJSON{
			ContainerJSONBase: &types.ContainerJSONBase{
				HostConfig: &containertypes.HostConfig{
					Resources: containertypes.Resources{
						CPUShares: 100,
					},
				},
				State: &types.ContainerState{
					StartedAt:  "2016-06-06T01:41:38.090313266Z",
					FinishedAt: "0001-01-01T00:00:00Z",
				},
			},
			Config: &containertypes.Config{},
			NetworkSettings: &types.NetworkSettings{
				Networks: nil,
			},
		}
	)

	engine := NewEngine("test", 0, engOpts)
	engine.setState(stateUnhealthy)
	assert.False(t, engine.isConnected())

	// A container is removed before it can be inspected.
	client := mockclient.NewMockClient()
	apiClient := engineapimock.NewMockClient()

	apiClient.On("Info", mock.Anything).Return(mockInfo, nil)
	apiClient.On("ServerVersion", mock.Anything).Return(mockVersion, nil)
	apiClient.On("NetworkList", mock.Anything,
		mock.AnythingOfType("NetworkListOptions"),
	).Return([]types.NetworkResource{}, nil)
	apiClient.On("VolumeList", mock.Anything,
		mock.AnythingOfType("Args"),
	).Return(volume.VolumesListOKBody{}, nil)
	apiClient.On("ImageList", mock.Anything, mock.AnythingOfType("ImageListOptions")).Return([]types.ImageSummary{}, nil)
	apiClient.On("Events", mock.Anything, mock.AnythingOfType("EventsOptions")).Return(make(chan events.Message), make(chan error))
	apiClient.On("NegotiateAPIVersion", mock.Anything).Return()

	apiClient.On(
		"ContainerList",
		mock.Anything,
		types.ContainerListOptions{
			All:  true,
			Size: false,
		},
	).Return(
		[]types.Container{container1, container2},
		nil,
	)

	apiClient.On("ContainerInspect", mock.Anything, "c1").Return(info1, errors.New("Not found"))
	apiClient.On("ContainerInspect", mock.Anything, "c2").Return(info2, nil)

	assert.NoError(t, engine.ConnectWithClient(client, apiClient))
	assert.Nil(t, engine.RefreshContainers(true))

	// List of containers is still valid
	containers := engine.Containers()
	assert.Len(t, containers, 1)
	assert.Equal(t, containers[0].ID, "c2")

	client.Mock.AssertExpectations(t)
	apiClient.Mock.AssertExpectations(t)
}

func TestDisconnect(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)

	client := mockclient.NewMockClient()
	apiClient := engineapimock.NewMockClient()

	apiClient.On("Info", mock.Anything).Return(mockInfo, nil)
	apiClient.On("ServerVersion", mock.Anything).Return(mockVersion, nil)
	apiClient.On("NetworkList", mock.Anything,
		mock.AnythingOfType("NetworkListOptions"),
	).Return([]types.NetworkResource{}, nil)
	apiClient.On("VolumeList", mock.Anything,
		mock.AnythingOfType("Args"),
	).Return(volume.VolumesListOKBody{}, nil)
	apiClient.On("Events", mock.Anything, mock.AnythingOfType("EventsOptions")).Return(make(chan events.Message), make(chan error))
	apiClient.On("NegotiateAPIVersion", mock.Anything).Return()

	// The client will return one container at first, then a second one will appear.
	apiClient.On("ImageList", mock.Anything, mock.AnythingOfType("ImageListOptions")).Return([]types.ImageSummary{}, nil)

	apiClient.On(
		"ContainerList",
		mock.Anything,
		types.ContainerListOptions{
			All:  true,
			Size: false,
		},
	).Return([]types.Container{{ID: "one"}}, nil).Once()

	apiClient.On(
		"ContainerInspect",
		mock.Anything,
		"one",
	).Return(
		types.ContainerJSON{
			ContainerJSONBase: &types.ContainerJSONBase{
				HostConfig: &containertypes.HostConfig{
					Resources: containertypes.Resources{
						CPUShares: 100,
					},
				},
				State: &types.ContainerState{
					StartedAt:  "2016-06-06T01:41:38.090313266Z",
					FinishedAt: "0001-01-01T00:00:00Z",
				},
			},
			Config: &containertypes.Config{},
			NetworkSettings: &types.NetworkSettings{
				Networks: nil,
			},
		},
		nil,
	).Once()

	filterArgs := filters.NewArgs()
	filterArgs.Add("id", "two")
	apiClient.On(
		"ContainerList",
		mock.Anything,
		types.ContainerListOptions{
			All:     true,
			Size:    false,
			Filters: filterArgs,
		},
	).Return(
		[]types.Container{{ID: "two"}},
		nil,
	).Once()

	apiClient.On("ContainerInspect", mock.Anything, "two").Return(
		types.ContainerJSON{
			ContainerJSONBase: &types.ContainerJSONBase{
				HostConfig: &containertypes.HostConfig{
					Resources: containertypes.Resources{
						CPUShares: 100,
					},
				},
				State: &types.ContainerState{
					StartedAt:  "2016-06-06T01:41:38.090313266Z",
					FinishedAt: "0001-01-01T00:00:00Z",
				},
			},
			Config: &containertypes.Config{},
			NetworkSettings: &types.NetworkSettings{
				Networks: nil,
			},
		},
		nil,
	).Once()

	assert.NoError(t, engine.ConnectWithClient(client, apiClient))
	assert.True(t, engine.isConnected())

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestDisconnect causes panic")
		}
	}()

	engine.Disconnect()
	assert.False(t, engine.isConnected())
	assert.True(t, engine.state == stateDisconnected)
	// Double disconnect shouldn't cause panic
	engine.Disconnect()
}

func TestRemoveImage(t *testing.T) {
	engine := NewEngine("test", 0, engOpts)

	imageName := "test-image"
	dIs := []types.ImageDeleteResponseItem{{Deleted: imageName}}

	apiClient := engineapimock.NewMockClient()
	apiClient.On("ImageList", mock.Anything, mock.AnythingOfType("ImageListOptions")).Return([]types.ImageSummary{}, nil)
	apiClient.On("ImageRemove", mock.Anything, mock.Anything,
		mock.AnythingOfType("ImageRemoveOptions")).Return(dIs, nil)
	engine.apiClient = apiClient

	deletedImages, err := engine.RemoveImage("test-image", true)
	if err != nil {
		t.Errorf("encountered an unexpected error")
	}
	if deletedImages[0].Deleted != imageName {
		t.Errorf("didn't get the image we removed")
	}
	apiClient.Mock.AssertExpectations(t)
}
