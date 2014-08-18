package libswarm

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

func (s *Server) OnLog(h func(...string) error) *Server {
	return s.OnVerb(Log, Handler(func(msg *Message) error {
		return h(msg.Args...)
	}))
}

func (s *Server) OnLs(h func() ([]string, error)) *Server {
	return s.OnVerb(Ls, Handler(func(msg *Message) error {
		names, err := h()
		if err != nil {
			return err
		}
		_, err = msg.Ret.Send(&Message{Verb: Set, Args: names})
		return err
	}))
}

func (s *Server) OnSpawn(h func(cmd ...string) (Sender, error)) *Server {
	return s.OnVerb(Spawn, Handler(func(msg *Message) error {
		obj, err := h(msg.Args...)
		if err != nil {
			return err
		}
		_, err = msg.Ret.Send(&Message{Verb: Ack, Ret: obj})
		return err
	}))
}

func (s *Server) OnAttach(h func(name string, ret Sender) error) *Server {
	return s.OnVerb(Attach, Handler(func(msg *Message) error {
		return h(msg.Args[0], msg.Ret)
	}))
}

func (s *Server) OnError(h func(...string) error) *Server {
	return s.OnVerb(Error, Handler(func(msg *Message) error {
		return h(msg.Args...)
	}))
}

func (s *Server) OnGet(h func() (string, error)) *Server {
	return s.OnVerb(Get, Handler(func(msg *Message) error {
		content, err := h()
		if err != nil {
			return err
		}
		_, err = msg.Ret.Send(&Message{Verb: Set, Args: []string{content}})
		return err
	}))
}

func (s *Server) OnStart(h func() error) *Server {
	return s.OnVerb(Start, Handler(func(msg *Message) error {
		if err := h(); err != nil {
			return err
		}
		_, err := msg.Ret.Send(&Message{Verb: Ack})
		return err
	}))
}

func (s *Server) OnStop(h func() error) *Server {
	return s.OnVerb(Stop, Handler(func(msg *Message) error {
		if err := h(); err != nil {
			return err
		}
		_, err := msg.Ret.Send(&Message{Verb: Ack})
		return err
	}))
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
