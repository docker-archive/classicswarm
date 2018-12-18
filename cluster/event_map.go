package cluster

import (
	"errors"
	"sync"

	log "github.com/sirupsen/logrus"
)

// ClusterEventHandlers is a map of EventHandler
type ClusterEventHandlers struct {
	sync.RWMutex
	eventHandlers map[EventHandler]struct{}
}

// NewClusterEventHandlers initializes and returns a ClusterEventHandlers object
func NewClusterEventHandlers() ClusterEventHandlers {
	return ClusterEventHandlers{
		eventHandlers: make(map[EventHandler]struct{}),
	}
}

// HandleAll callbacks for the events
func (eh *ClusterEventHandlers) Handle(e *Event) error {
	eh.RLock()
	defer eh.RUnlock()

	for h := range eh.eventHandlers {
		if err := h.Handle(e); err != nil {
			log.Error(err)
		}
	}
	return nil
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

// CloseWatchQueues unregisters all API event handlers (the ones with
// watch queues) and closes the respective queues. This should be
// called when the manager shuts down
func (eh *ClusterEventHandlers) CloseWatchQueues() {
	eh.Lock()
	defer eh.Unlock()

	for h := range eh.eventHandlers {
		// only do this for API event handlers
		apiHandler, ok := h.(*APIEventHandler)
		if !ok {
			continue
		}
		apiHandler.cleanupHandler()
		delete(eh.eventHandlers, h)
	}
}
