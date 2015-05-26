package mesos

import (
	"strings"
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

const name = "mesos-swarm-task-name"

func TestBuild(t *testing.T) {
	task, err := newTask(nil, cluster.BuildContainerConfig(dockerclient.ContainerConfig{
		Image:     "test-image",
		CpuShares: 42,
		Memory:    2097152,
		Cmd:       []string{"ls", "foo", "bar"},
	}), name)
	assert.NoError(t, err)

	task.build("slave-id")

	assert.Equal(t, task.Container.GetType(), mesosproto.ContainerInfo_DOCKER)
	assert.Equal(t, task.Container.Docker.GetImage(), "test-image")
	assert.Equal(t, task.Container.Docker.GetNetwork(), mesosproto.ContainerInfo_DockerInfo_BRIDGE)

	assert.Equal(t, len(task.Resources), 2)
	assert.Equal(t, task.Resources[0], mesosutil.NewScalarResource("cpus", 42.0))
	assert.Equal(t, task.Resources[1], mesosutil.NewScalarResource("mem", 2))

	assert.Equal(t, task.Command.GetValue(), "ls")
	assert.Equal(t, task.Command.GetArguments(), []string{"foo", "bar"})

	assert.Equal(t, len(task.Container.Docker.GetParameters()), 1)
	assert.Equal(t, task.Container.Docker.GetParameters()[0].GetKey(), "label")
	assert.Equal(t, task.Container.Docker.GetParameters()[0].GetValue(), "com.docker.swarm.mesos.name="+name)

	assert.Equal(t, task.SlaveId.GetValue(), "slave-id")
}

func TestNewTask(t *testing.T) {
	task, err := newTask(nil, cluster.BuildContainerConfig(dockerclient.ContainerConfig{}), name)
	assert.NoError(t, err)

	assert.Equal(t, *task.Name, name)
	assert.True(t, strings.HasPrefix(task.TaskId.GetValue(), name+"."))
	assert.Equal(t, len(task.TaskId.GetValue()), len(name)+1+12) //<name>+.+<shortId>
}

func TestSendGetStatus(t *testing.T) {
	task, err := newTask(nil, cluster.BuildContainerConfig(dockerclient.ContainerConfig{}), "")
	assert.NoError(t, err)

	status := mesosutil.NewTaskStatus(nil, mesosproto.TaskState_TASK_RUNNING)

	go func() { task.sendStatus(status) }()
	s := task.getStatus()

	assert.Equal(t, s, status)
}
