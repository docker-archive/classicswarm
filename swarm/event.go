package swarm

import "time"

type Event struct {
	Type      string
	Container *Container
	Node      *Node
	Time      time.Time
}

type EventHandler interface {
	Handle(*Event) error
}
