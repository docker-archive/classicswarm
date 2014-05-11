package inmem

import (
	"container/list"
	"sync"
)

// StackSender forwards beam messages to a dynamic list of backend receivers.
// New backends are stacked on top. When a message is sent, each backend is
// tried until one succeeds. Any failing backends encountered along the way 
// are removed from the queue.
type StackSender struct {
	stack *list.List
	l     sync.RWMutex
}

func NewStackSender() *StackSender {
	stack := list.New()
	return &StackSender{
		stack: stack,
	}
}

func (s *StackSender) Send(msg *Message, mode int) (Receiver, Sender, error) {
	s.l.RLock()
	e := s.stack.Front()
	s.l.RUnlock()
	for e != nil {
		h := e.Value.(Sender)
		r, w, err := h.Send(msg, mode)
		if err == nil {
			return r, w, nil
		}
		s.l.Lock()
		next := e.Next()
		s.stack.Remove(e)
		s.l.Unlock()
		e = next
	}
	// Silently drop messages if no valid backend is available.
	return nil, nil, nil
}

func (s *StackSender) Add(dst Sender) {
	s.l.Lock()
	defer s.l.Unlock()
	s.stack.PushFront(dst)
}
