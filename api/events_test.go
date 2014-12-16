package api

import (
	"fmt"
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/stretchr/testify/assert"
)

type FakeWriter struct {
	Tmp []byte
}

func (fw *FakeWriter) Write(p []byte) (n int, err error) {
	fw.Tmp = append(fw.Tmp, p...)
	return len(p), nil
}

func TestHandle(t *testing.T) {
	eh := NewEventsHandler()
	assert.Equal(t, eh.Size(), 0)

	fw := &FakeWriter{Tmp: []byte{}}
	eh.Add("test", fw)

	assert.Equal(t, eh.Size(), 1)

	event := &cluster.Event{
		Node: &cluster.Node{
			Name: "node_name",
			ID:   "node_id",
			Addr: "node_addr",
			IP:   "node_ip",
		},
	}
	event.Event.Status = "status"
	event.Event.Id = "id"
	event.Event.From = "from"
	event.Event.Time = 0

	assert.NoError(t, eh.Handle(event))

	str := fmt.Sprintf("{%q:%q,%q:%q,%q:%q,%q:%d,%q:%q,%q:%q,%q:%q,%q:%q}",
		"status", "status",
		"id", "id",
		"from", "from node:node_name",
		"time", 0,
		"node_name", "node_name",
		"node_id", "node_id",
		"node_addr", "node_addr",
		"node_ip", "node_ip")

	assert.Equal(t, str, string(fw.Tmp))
}
