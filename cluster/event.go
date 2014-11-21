package cluster

import "github.com/samalba/dockerclient"

type Event struct {
	dockerclient.Event

	NodeName string
	NodeID   string
	NodeAddr string
	NodeIP   string
}

type EventHandler interface {
	Handle(*Event) error
}
