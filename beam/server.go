package beam

import (
	"github.com/docker/libchan"

	"fmt"
)

type Server struct {
	handlers map[Verb]Sender
	catchall Sender
}

func NewServer() *Server {
	return &Server{
		handlers: make(map[Verb]Sender),
	}
}

func (s *Server) Catchall(h Sender) *Server {
	s.catchall = h
	return s
}

func (s *Server) OnVerb(v Verb, h Sender) *Server {
	s.handlers[v] = h
	return s
}

func (s *Server) Send(msg *Message) (Receiver, error) {
	if h, exists := s.handlers[msg.Verb]; exists {
		return h.Send(msg)
	}
	if s.catchall != nil {
		return s.catchall.Send(msg)
	}
	return NotImplemented.Send(msg)
}

func (s *Server) Close() error {
	return fmt.Errorf("can't close")
}

func (s *Server) Unwrap() libchan.Sender {
	return &senderUnwrapper{s}
}
