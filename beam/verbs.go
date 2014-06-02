package beam

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
)

type Verb string

var (
	Ack    Verb = "ack"
	Log    Verb = "log"
	Start  Verb = "start"
	Stop   Verb = "stop"
	Attach Verb = "attach"
	Spawn  Verb = "spawn"
	Set    Verb = "set"
	Get    Verb = "get"
	File   Verb = "file"
	Error  Verb = "error"
	Ls     Verb = "ls"
)

var NotImplemented = Repeater(&Message{Name: "error", Args: []string{"not implemented"}})

type Handler func(msg *Message) error

func (h Handler) Send(msg *Message) (Receiver, error) {
	var ret Receiver
	if RetPipe.Equals(msg.Ret) {
		ret, msg.Ret = Pipe()
	}
	go func() {
		// Ret must always be a valid Sender, so handlers
		// can safely send to it
		if msg.Ret == nil {
			msg.Ret = NopSender{}
		}
		err := h(msg)
		if err != nil {
			Obj(msg.Ret).Error("%v", err)
		}
		msg.Ret.Close()
	}()
	return ret, nil
}

func (h Handler) Close() error {
	return fmt.Errorf("can't close")
}

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

func (s *Server) Send(msg *Message) (Receiver, error) {
	if h, exists := s.handlers[Verb(msg.Name)]; exists {
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

// FIXME: rename Object to Client

type Object struct {
	Sender
}

func Obj(dst Sender) *Object {
	return &Object{dst}
}

func (o *Object) Log(msg string, args ...interface{}) error {
	_, err := o.Send(&Message{Name: "log", Args: []string{fmt.Sprintf(msg, args...)}})
	return err
}

func (o *Object) Ls() ([]string, error) {
	ret, err := o.Send(&Message{Name: "ls", Ret: RetPipe})
	if err != nil {
		return nil, err
	}
	msg, err := ret.Receive(0)
	if err == io.EOF {
		return nil, fmt.Errorf("unexpected EOF")
	}
	if msg.Name == "set" {
		if err != nil {
			return nil, err
		}
		return msg.Args, nil
	}
	if msg.Name == "error" {
		return nil, fmt.Errorf(strings.Join(msg.Args[:1], ""))
	}
	return nil, fmt.Errorf("unexpected verb %v", msg.Name)
}

func (o *Object) Spawn(cmd ...string) (out *Object, err error) {
	ret, err := o.Send(&Message{Name: "spawn", Args: cmd, Ret: RetPipe})
	if err != nil {
		return nil, err
	}
	msg, err := ret.Receive(Ret)
	if err == io.EOF {
		return nil, fmt.Errorf("unexpected EOF")
	}
	if err != nil {
		return nil, err
	}
	if msg.Name == "ack" {
		return &Object{msg.Ret}, nil
	}
	msg.Ret.Close()
	if msg.Name == "error" {
		return nil, fmt.Errorf("%s", strings.Join(msg.Args[:1], ""))
	}
	return nil, fmt.Errorf("unexpected verb %v", msg.Name)
}

func (o *Object) Attach(name string) (in Receiver, out *Object, err error) {
	ret, err := o.Send(&Message{Name: "attach", Args: []string{name}, Ret: RetPipe})
	if err != nil {
		return nil, nil, err
	}
	msg, err := ret.Receive(Ret)
	if err == io.EOF {
		return nil, nil, fmt.Errorf("unexpected EOF")
	}
	if err != nil {
		return nil, nil, err
	}
	if msg.Name == "ack" {
		return ret, &Object{msg.Ret}, nil
	}
	msg.Ret.Close()
	if msg.Name == "error" {
		return nil, nil, fmt.Errorf("%s", strings.Join(msg.Args[:1], ""))
	}
	return nil, nil, fmt.Errorf("unexpected verb %v", msg.Name)
}

func (o *Object) Error(msg string, args ...interface{}) error {
	_, err := o.Send(&Message{Name: "error", Args: []string{fmt.Sprintf(msg, args...)}})
	return err
}

func (o *Object) Connect() (net.Conn, error) {
	ret, err := o.Send(&Message{Name: "connect", Ret: RetPipe})
	if err != nil {
		return nil, err
	}
	// FIXME: set Att
	msg, err := ret.Receive(0)
	if err == io.EOF {
		return nil, fmt.Errorf("unexpected EOF")
	}
	if msg.Name == "connect" {
		if msg.Att == nil {
			return nil, fmt.Errorf("missing attachment")
		}
		conn, err := net.FileConn(msg.Att)
		if err != nil {
			msg.Att.Close()
			return nil, err
		}
		msg.Att.Close()
		return conn, nil
	}
	if msg.Name == "error" {
		return nil, fmt.Errorf(strings.Join(msg.Args[:1], ""))
	}
	return nil, fmt.Errorf("unexpected verb %v", msg.Name)
}

func (o *Object) SetJson(val interface{}) error {
	txt, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return o.Set(string(txt))
}

func (o *Object) Set(vals ...string) error {
	_, err := o.Send(&Message{Name: "set", Args: vals})
	return err
}

func (o *Object) Get(key string) (string, error) {
	ret, err := o.Send(&Message{Name: "get", Args: []string{key}, Ret: RetPipe})
	if err != nil {
		return "", err
	}
	msg, err := ret.Receive(0)
	if err == io.EOF {
		return "", fmt.Errorf("unexpected EOF")
	}
	if msg.Name == "set" {
		if err != nil {
			return "", err
		}
		if len(msg.Args) != 1 {
			return "", fmt.Errorf("protocol error")
		}
		return msg.Args[0], nil
	}
	if msg.Name == "error" {
		return "", fmt.Errorf(strings.Join(msg.Args[:1], ""))
	}
	return "", fmt.Errorf("unexpected verb %v", msg.Name)
}

func (o *Object) Watch() (Receiver, error) {
	ret, err := o.Send(&Message{Name: "watch", Ret: RetPipe})
	if err != nil {
		return nil, err
	}
	msg, err := ret.Receive(0)
	if msg.Name == "ok" {
		return ret, nil
	}
	if msg.Name == "error" {
		return nil, fmt.Errorf(strings.Join(msg.Args[:1], ""))
	}
	return nil, fmt.Errorf("unexpected verb %v", msg.Name)
}

func (o *Object) Start() error {
	ret, err := o.Send(&Message{Name: "start", Ret: RetPipe})
	msg, err := ret.Receive(0)
	if err == io.EOF {
		return fmt.Errorf("unexpected EOF")
	}
	if Verb(msg.Name) == Ack {
		return nil
	}
	if msg.Name == "error" {
		return fmt.Errorf(strings.Join(msg.Args[:1], ""))
	}
	return fmt.Errorf("unexpected verb %v", msg.Name)
}
