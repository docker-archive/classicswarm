package task

import (
	"sort"
	"strings"
	"testing"
	"time"

	containertypes "github.com/docker/docker/api/types/container"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/swarm/cluster"
	"github.com/gogo/protobuf/proto"
	"github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
	"github.com/stretchr/testify/assert"
)

const name = "mesos-swarm-task-name"

var (
	containerConfig = containertypes.Config{
		Image: "test-image",
		Cmd:   []string{"ls", "foo", "bar"},
	}
	hostConfig = containertypes.HostConfig{
		Resources: containertypes.Resources{
			CPUShares: 42,
			Memory:    2097152,
		},
	}
	networkingConfig = networktypes.NetworkingConfig{}
)

func TestBuild(t *testing.T) {
	task, err := NewTask(cluster.BuildContainerConfig(containerConfig, hostConfig, networkingConfig), name, 5*time.Second)
	assert.NoError(t, err)

	var resources []*mesosproto.Resource
	defaultRole := "*"
	resource := &mesosproto.Resource{
		Name:   proto.String("cpus"),
		Type:   mesosproto.Value_SCALAR.Enum(),
		Scalar: &mesosproto.Value_Scalar{Value: proto.Float64(42)},
		Role:   &defaultRole,
	}
	resources = append(resources, resource)

	resource = &mesosproto.Resource{
		Name:   proto.String("mem"),
		Type:   mesosproto.Value_SCALAR.Enum(),
		Scalar: &mesosproto.Value_Scalar{Value: proto.Float64(2097152)},
		Role:   &defaultRole,
	}
	resources = append(resources, resource)

	offer := &mesosproto.Offer{
		Resources: resources,
	}

	offers := make(map[string]*mesosproto.Offer)
	offers["test"] = offer

	task.Build("slave-id", offers, "*")

	assert.Equal(t, task.Container.GetType(), mesosproto.ContainerInfo_DOCKER)
	assert.Equal(t, task.Container.Docker.GetImage(), "test-image")
	assert.Equal(t, task.Container.Docker.GetNetwork(), mesosproto.ContainerInfo_DockerInfo_BRIDGE)

	assert.Equal(t, len(task.Resources), 2)
	resource = mesosutil.NewScalarResource("cpus", 42.0)
	resource.Role = &defaultRole
	assert.Equal(t, task.Resources[0], resource)
	resource = mesosutil.NewScalarResource("mem", 2)
	resource.Role = &defaultRole
	assert.Equal(t, task.Resources[1], resource)

	assert.Equal(t, task.Command.GetValue(), "ls")
	assert.Equal(t, task.Command.GetArguments(), []string{"foo", "bar"})

	parameters := []string{task.Container.Docker.GetParameters()[0].GetValue(), task.Container.Docker.GetParameters()[1].GetValue()}
	sort.Strings(parameters)

	assert.Equal(t, len(parameters), 2)
	assert.Equal(t, parameters[0], "com.docker.swarm.mesos.name="+name)
	assert.Equal(t, parameters[1], "com.docker.swarm.mesos.task="+*task.TaskId.Value)

	assert.Equal(t, task.SlaveId.GetValue(), "slave-id")
}

func TestNewTask(t *testing.T) {
	task, err := NewTask(cluster.BuildContainerConfig(containertypes.Config{}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}), name, 5*time.Second)
	assert.NoError(t, err)

	assert.Equal(t, *task.Name, name)
	assert.True(t, strings.HasPrefix(task.TaskId.GetValue(), name+"."))
	assert.Equal(t, len(task.TaskId.GetValue()), len(name)+1+12) //<name>+.+<shortId>
}

func TestSendGetStatus(t *testing.T) {
	task, err := NewTask(cluster.BuildContainerConfig(containertypes.Config{}, containertypes.HostConfig{}, networktypes.NetworkingConfig{}), "", 5*time.Second)
	assert.NoError(t, err)

	status := mesosutil.NewTaskStatus(nil, mesosproto.TaskState_TASK_RUNNING)

	go func() { task.SendStatus(status) }()
	s := task.GetStatus()

	assert.Equal(t, s, status)
}
