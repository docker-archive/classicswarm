package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type item struct {
	id    string
	count int
}

func (i *item) ID() string {
	return i.id
}

func (i *item) Do() bool {
	i.count = i.count - 1
	return i.count == 0
}

func TestAdd(t *testing.T) {
	q := NewQueue()

	q.Add(&item{"id1", 1})
	assert.Equal(t, len(q.items), 0)

	q.Add(&item{"id2", 2})
	assert.Equal(t, len(q.items), 1)

}

func TestRemove(t *testing.T) {
	q := NewQueue()

	i := &item{"id1", 2}
	q.Add(i)
	assert.Equal(t, len(q.items), 1)
	q.Remove(i)
	assert.Equal(t, len(q.items), 0)

}

func TestProcess(t *testing.T) {
	q := NewQueue()

	q.Add(&item{"id1", 2})
	assert.Equal(t, len(q.items), 1)
	q.Process()
	assert.Equal(t, len(q.items), 0)

	q.Add(&item{"id2", 3})
	assert.Equal(t, len(q.items), 1)
	q.Process()
	assert.Equal(t, len(q.items), 1)
	q.Process()
	assert.Equal(t, len(q.items), 0)

}
