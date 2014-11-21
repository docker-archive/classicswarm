package cluster

import "github.com/samalba/dockerclient"

type Event struct {
	dockerclient.Event

	NodeName string
}

type EventHandler interface {
	Handle(*Event) error
}
