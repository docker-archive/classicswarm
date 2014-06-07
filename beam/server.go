package beam

import (
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

func (s *Server) OnSpawn(h Sender) *Server {
	return s.OnVerb(Spawn, h)
}

func (s *Server) OnStart(h Sender) *Server {
	return s.OnVerb(Start, h)
}

func (s *Server) OnStop(h Sender) *Server {
	return s.OnVerb(Stop, h)
}

func (s *Server) OnAttach(h Sender) *Server {
	return s.OnVerb(Attach, h)
}

func (s *Server) OnLog(h Sender) *Server {
	return s.OnVerb(Log, h)
}

func (s *Server) OnError(h Sender) *Server {
	return s.OnVerb(Error, h)
}

func (s *Server) OnLs(h Sender) *Server {
	return s.OnVerb(Ls, h)
}

func (s *Server) OnGet(h Sender) *Server {
	return s.OnVerb(Get, h)
}

func (s *Server) OnGetChildren(h Sender) *Server {
	return s.OnVerb(GetChildren, h)
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
