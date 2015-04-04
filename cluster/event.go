package cluster

import "github.com/samalba/dockerclient"

// Event is exported
type Event struct {
	dockerclient.Event
	Engine *Engine
}

// EventHandler is exported
type EventHandler interface {
	Handle(*Event) error
}
