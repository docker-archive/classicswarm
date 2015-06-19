package mesos

import (
	"testing"

	"github.com/mesos/mesos-go/mesosproto"
	"github.com/stretchr/testify/assert"
)

type MockHandler struct{}

func (h MockHandler) Process(ts []*task) []*task {
	return ts
}

func newMockQueue() *Queue {
	q := &Queue{}
	q.tasks = make(map[string]*task)
	return q
}

func mockTask(id string) *task {
	t := &task{}
	t.TaskId = &mesosproto.TaskID{Value: &id}
	return t
}

func TestAdd(t *testing.T) {
	q := newMockQueue()

	q.Add(mockTask("1"))
	assert.Equal(t, 1, len(q.tasks))

	q.Add(mockTask("2"))
	assert.Equal(t, 2, len(q.tasks))
}

func TestRemove(t *testing.T) {
	q := newMockQueue()

	i := mockTask("1")
	q.Add(i)
	assert.Equal(t, len(q.tasks), 1)
	q.Remove(i)
	assert.Equal(t, len(q.tasks), 0)
}

func TestProcess(t *testing.T) {
	q := newMockQueue()
	h := MockHandler{}

	q.Add(mockTask("1"))
	assert.Equal(t, len(q.tasks), 1)
	q.Process(h)
	assert.Equal(t, len(q.tasks), 0)

	q.Add(mockTask("2"))
	assert.Equal(t, len(q.tasks), 1)
	q.Process(h)
	assert.Equal(t, len(q.tasks), 0)
}
