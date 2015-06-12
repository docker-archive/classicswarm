package utils

import (
	"github.com/samalba/dockerclient"
	"github.com/samalba/dockerclient/mockclient"
	"github.com/stretchr/testify/mock"
)

// MockEvent mock a docker event emitter for testing.
type MockEvent struct {
	retCh  chan dockerclient.EventOrError
	buf    chan dockerclient.EventOrError
	stopCh chan struct{}
}

// NewMockEvent create a new instance of MockEvent and stub MonitorEvents API.
func NewMockEvent(client *mockclient.MockClient, stopCh chan struct{}) *MockEvent {
	m := &MockEvent{
		retCh:  make(chan dockerclient.EventOrError),
		buf:    make(chan dockerclient.EventOrError),
		stopCh: stopCh,
	}
	func(ret <-chan dockerclient.EventOrError) {
		client.On("MonitorEvents", mock.Anything, mock.Anything).Return(ret, nil)
	}(m.retCh)
	go m.dispatchEvents()
	return m
}

// Emit an event for testing.
func (m *MockEvent) Emit(ev dockerclient.EventOrError) {
	m.buf <- ev
}

func (m *MockEvent) dispatchEvents() {
	for {
		select {
		case <-m.stopCh:
			return
		case m.retCh <- <-m.buf:
		}
	}
}
