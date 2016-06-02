package cluster

import (
	"errors"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types/events"
)

// Event is exported
type Event struct {
	events.Message
	Engine *Engine `json:"-"`
}

// EventHandler is exported
type EventHandler interface {
	Handle(*Event) error
}

// EventHandlers is a map of EventHandler
type EventHandlers struct {
	sync.RWMutex

	eventHandlers map[EventHandler]struct{}
}

// NewEventHandlers returns an EventHandlers
func NewEventHandlers() *EventHandlers {
	return &EventHandlers{
		eventHandlers: make(map[EventHandler]struct{}),
	}
}

// Handle callbacks for the events
func (eh *EventHandlers) Handle(e *Event) {
	eh.RLock()
	defer eh.RUnlock()

	for h := range eh.eventHandlers {
		if err := h.Handle(e); err != nil {
			log.Error(err)
		}
	}
}

// RegisterEventHandler registers an event handler.
func (eh *EventHandlers) RegisterEventHandler(h EventHandler) error {
	eh.Lock()
	defer eh.Unlock()

	if _, ok := eh.eventHandlers[h]; ok {
		return errors.New("event handler already set")
	}
	eh.eventHandlers[h] = struct{}{}
	return nil
}

// UnregisterEventHandler unregisters a previously registered event handler.
func (eh *EventHandlers) UnregisterEventHandler(h EventHandler) {
	eh.Lock()
	defer eh.Unlock()

	delete(eh.eventHandlers, h)
}
