package cluster

import (
	"errors"
	"sync"

	log "github.com/Sirupsen/logrus"
)

// ClusterEventHandlers is a map of EventHandler
type ClusterEventHandlers struct {
	sync.RWMutex
	eventHandlers map[EventHandler]struct{}
}

// NewClusterEventHandlers returns an EventHandlers
func NewClusterEventHandlers() *ClusterEventHandlers {
	return &ClusterEventHandlers{
		eventHandlers: make(map[EventHandler]struct{}),
	}
}

// Handle callbacks for the events
func (eh *ClusterEventHandlers) Handle(e *Event) {
	eh.RLock()
	defer eh.RUnlock()

	for h := range eh.eventHandlers {
		if err := h.Handle(e); err != nil {
			log.Error(err)
		}
	}
}

// RegisterEventHandler registers an event handler.
func (eh *ClusterEventHandlers) RegisterEventHandler(h EventHandler) error {
	eh.Lock()
	defer eh.Unlock()

	if _, ok := eh.eventHandlers[h]; ok {
		return errors.New("event handler already set")
	}
	eh.eventHandlers[h] = struct{}{}
	return nil
}

// UnregisterEventHandler unregisters a previously registered event handler.
func (eh *ClusterEventHandlers) UnregisterEventHandler(h EventHandler) {
	eh.Lock()
	defer eh.Unlock()

	delete(eh.eventHandlers, h)
}
