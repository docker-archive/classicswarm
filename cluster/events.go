package cluster

import "github.com/docker/docker/api/types/events"

// Event is exported
type Event struct {
	events.Message
	Engine *Engine `json:"-"`
}

// EventHandler is exported
type EventHandler interface {
	Handle(*Event) error
}

// EventHandler is implemented by all event handlers in Swarm
// - APIEventHandler: Handles API level events
// - Watchdog: Handles events related to rescheduling
// - Cluster: Acts as a proxy event handler for the engine, but essentially
// punts all handling to the above two handlers
