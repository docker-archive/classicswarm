package inmem

import (
	"container/list"
	"fmt"
	"sync"
)

// StackSender forwards beam messages to a dynamic list of backend receivers.
// New backends are stacked on top. When a message is sent, each backend is
// tried until one succeeds. Any failing backends encountered along the way
// are removed from the queue.
type StackSender struct {
	stack *list.List
	front *list.Element
	l     sync.RWMutex
}

func NewStackSender() *StackSender {
	stack := list.New()
	return &StackSender{
		stack: stack,
	}
}

func (s *StackSender) Send(msg *Message, mode int) (r Receiver, w Sender, err error) {
	ok := s.walk(func(h Sender) (ok bool) {
		r, w, err = h.Send(msg, mode)
		fmt.Printf("sending %v to %#v returned %v\n", msg, h, err)
		if err == nil {
			return true
		}
		return false
	})
	if ok {
		return r, w, err
	}
	// Silently drop messages if no valid backend is available.
	return nil, nil, nil
}

func (s *StackSender) Add(dst Sender) *StackSender {
	s.l.Lock()
	defer s.l.Unlock()
	prev := &StackSender{
		stack: s.stack,
		front: s.stack.Front(),
	}
	s.stack.PushFront(dst)
	return prev
}

func (s *StackSender) Close() error {
	s.walk(func(h Sender) bool {
		h.Close()
		// remove all handlers
		return false
	})
	return nil
}

func (s *StackSender) walk(f func(Sender) bool) bool {
	var e *list.Element
	if s.front == nil {
		s.l.RLock()
		e = s.stack.Front()
		s.l.RUnlock()
	} else {
		e = s.front
	}
	for e != nil {
		fmt.Printf("[StackSender.Walk] %v\n", e.Value.(Sender))
		ok := f(e.Value.(Sender))
		if ok {
			fmt.Printf("[StackSender.Walk] %v -> OK\n", e.Value.(Sender))
			return true
		}
		fmt.Printf("[StackSender.Walk] %v invalid: removing\n", e.Value.(Sender))
		s.l.Lock()
		next := e.Next()
		s.stack.Remove(e)
		s.l.Unlock()
		e = next
	}
	fmt.Printf("[STackSender.Walk] -> no valid handler\n")
	return false
}
func (s *StackSender) Len() int {
	s.l.RLock()
	defer s.l.RUnlock()
	if s.front == nil {
		return s.stack.Len()
	}
	var len int
	for e := s.front; e != nil; e = e.Next() {
		len++
	}
	return len
}
