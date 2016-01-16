package api

import (
	"encoding/json"
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
	eh := newEventsHandler()
	assert.Equal(t, eh.Size(), 0)

	fw := &FakeWriter{Tmp: []byte{}}
	eh.Add("test", fw)

	assert.Equal(t, eh.Size(), 1)

	event := &cluster.Event{
		Engine: &cluster.Engine{
			ID:   "node_id",
			Name: "node_name",
			IP:   "node_ip",
			Addr: "node_addr",
		},
	}

	event.Event.Status = "status"
	event.Event.ID = "id"
	event.Event.From = "from"
	event.Event.Time = 0
	event.Actor.Attributes = make(map[string]string)
	event.Actor.Attributes["nodevent.name"] = event.Engine.Name
	event.Actor.Attributes["nodevent.id"] = event.Engine.ID
	event.Actor.Attributes["nodevent.addr"] = event.Engine.Addr
	event.Actor.Attributes["nodevent.ip"] = event.Engine.IP

	assert.NoError(t, eh.Handle(event))

	event.Event.From = "from node:node_name"

	data, err := json.Marshal(event)
	assert.NoError(t, err)

	node := fmt.Sprintf(",%q:{%q:%q,%q:%q,%q:%q,%q:%q}}",
		"node",
		"Name", event.Engine.Name,
		"Id", event.Engine.ID,
		"Addr", event.Engine.Addr,
		"Ip", event.Engine.IP,
	)

	// insert Node field
	data = data[:len(data)-1]
	data = append(data, []byte(node)...)

	assert.Equal(t, string(data), string(fw.Tmp))
}
