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

type FakeNode struct{}

func (fn *FakeNode) ID() string                            { return "node_id" }
func (fn *FakeNode) Name() string                          { return "node_name" }
func (fn *FakeNode) IP() string                            { return "node_ip" }
func (fn *FakeNode) Addr() string                          { return "node_addr" }
func (fn *FakeNode) Images() []*cluster.Image              { return nil }
func (fn *FakeNode) Image(_ string) *cluster.Image         { return nil }
func (fn *FakeNode) Containers() []*cluster.Container      { return nil }
func (fn *FakeNode) Container(_ string) *cluster.Container { return nil }
func (fn *FakeNode) TotalCpus() int64                      { return 0 }
func (fn *FakeNode) UsedCpus() int64                       { return 0 }
func (fn *FakeNode) TotalMemory() int64                    { return 0 }
func (fn *FakeNode) UsedMemory() int64                     { return 0 }
func (fn *FakeNode) Labels() map[string]string             { return nil }
func (fn *FakeNode) IsHealthy() bool                       { return true }

func TestHandle(t *testing.T) {
	eh := NewEventsHandler()
	assert.Equal(t, eh.Size(), 0)

	fw := &FakeWriter{Tmp: []byte{}}
	eh.Add("test", fw)

	assert.Equal(t, eh.Size(), 1)

	event := &cluster.Event{
		Node: &FakeNode{},
	}

	event.Event.Status = "status"
	event.Event.Id = "id"
	event.Event.From = "from"
	event.Event.Time = 0

	assert.NoError(t, eh.Handle(event))

	str := fmt.Sprintf("{%q:%q,%q:%q,%q:%q,%q:%d,%q:{%q:%q,%q:%q,%q:%q,%q:%q}}",
		"status", "status",
		"id", "id",
		"from", "from node:node_name",
		"time", 0,
		"node",
		"Name", "node_name",
		"Id", "node_id",
		"Addr", "node_addr",
		"Ip", "node_ip")

	assert.Equal(t, str, string(fw.Tmp))
}
