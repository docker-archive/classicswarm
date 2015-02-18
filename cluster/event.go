package cluster

import "github.com/samalba/dockerclient"

type Event struct {
	dockerclient.Event
	Node Node
}

type EventHandler interface {
	Handle(*Event) error
}
