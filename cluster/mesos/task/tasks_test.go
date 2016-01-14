package task

import (
	"testing"
	"time"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

type testLauncher struct {
	count int
}

func (t *testLauncher) LaunchTask(_ *Task) bool {
	t.count = t.count - 1
	return t.count == 0
}

func TestAdd(t *testing.T) {
	q := NewTasks(&testLauncher{count: 1})

	task1, _ := NewTask(cluster.BuildContainerConfig(dockerclient.ContainerConfig{
		Image:     "test-image",
		CpuShares: 42,
		Memory:    2097152,
		Cmd:       []string{"ls", "foo", "bar"},
	}), "name1", 5*time.Second)

	task2, _ := NewTask(cluster.BuildContainerConfig(dockerclient.ContainerConfig{
		Image:     "test-image",
		CpuShares: 42,
		Memory:    2097152,
		Cmd:       []string{"ls", "foo", "bar"},
	}), "name2", 5*time.Second)
	q.Add(task1)
	assert.Equal(t, len(q.Tasks), 0)

	q.Add(task2)
	assert.Equal(t, len(q.Tasks), 1)

}

func TestRemove(t *testing.T) {
	q := NewTasks(&testLauncher{count: 2})
	task1, _ := NewTask(cluster.BuildContainerConfig(dockerclient.ContainerConfig{
		Image:     "test-image",
		CpuShares: 42,
		Memory:    2097152,
		Cmd:       []string{"ls", "foo", "bar"},
	}), "name1", 5*time.Second)

	q.Add(task1)
	assert.Equal(t, len(q.Tasks), 1)
	q.Remove(task1)
	assert.Equal(t, len(q.Tasks), 0)

}

func TestProcess(t *testing.T) {
	q := NewTasks(&testLauncher{count: 3})
	task1, _ := NewTask(cluster.BuildContainerConfig(dockerclient.ContainerConfig{
		Image:     "test-image",
		CpuShares: 42,
		Memory:    2097152,
		Cmd:       []string{"ls", "foo", "bar"},
	}), "name1", 5*time.Second)

	q.Add(task1)
	assert.Equal(t, len(q.Tasks), 1)
	q.Process()
	assert.Equal(t, len(q.Tasks), 1)
	q.Process()
	assert.Equal(t, len(q.Tasks), 0)

}
