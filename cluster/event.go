package cluster

import "github.com/samalba/dockerclient"

// Event is exported
type Event struct {
	dockerclient.Event
	Node Node
}

// EventHandler is exported
type EventHandler interface {
	Handle(*Event) error
}
