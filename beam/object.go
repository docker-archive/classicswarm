package beam

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
)

// FIXME: rename Object to Client

type Object struct {
	Sender
}

func Obj(dst Sender) *Object {
	return &Object{dst}
}

func (o *Object) Log(msg string, args ...interface{}) error {
	_, err := o.Send(&Message{Verb: Log, Args: []string{fmt.Sprintf(msg, args...)}})
	return err
}

func (o *Object) Ls() ([]string, error) {
	ret, err := o.Send(&Message{Verb: Ls, Ret: RetPipe})
	if err != nil {
		return nil, err
	}
	msg, err := ret.Receive(0)
	if err == io.EOF {
		return nil, fmt.Errorf("unexpected EOF")
	}
	if msg.Verb == Set {
		if err != nil {
			return nil, err
		}
		return msg.Args, nil
	}
	if msg.Verb == Error {
		return nil, fmt.Errorf(strings.Join(msg.Args[:1], ""))
	}
	return nil, fmt.Errorf("unexpected verb %v", msg.Verb)
}

func (o *Object) Spawn(cmd ...string) (out *Object, err error) {
	ret, err := o.Send(&Message{Verb: Spawn, Args: cmd, Ret: RetPipe})
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
	if msg.Verb == Ack {
		return &Object{msg.Ret}, nil
	}
	msg.Ret.Close()
	if msg.Verb == Error {
		return nil, fmt.Errorf("%s", strings.Join(msg.Args[:1], ""))
	}
	return nil, fmt.Errorf("unexpected verb %v", msg.Verb)
}

func (o *Object) Attach(name string) (in Receiver, out *Object, err error) {
	ret, err := o.Send(&Message{Verb: Attach, Args: []string{name}, Ret: RetPipe})
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
	if msg.Verb == Ack {
		return ret, &Object{msg.Ret}, nil
	}
	msg.Ret.Close()
	if msg.Verb == Error {
		return nil, nil, fmt.Errorf("%s", strings.Join(msg.Args[:1], ""))
	}
	return nil, nil, fmt.Errorf("unexpected verb %v", msg.Verb)
}

func (o *Object) Error(msg string, args ...interface{}) error {
	_, err := o.Send(&Message{Verb: Error, Args: []string{fmt.Sprintf(msg, args...)}})
	return err
}

func (o *Object) Connect() (net.Conn, error) {
	ret, err := o.Send(&Message{Verb: Connect, Ret: RetPipe})
	if err != nil {
		return nil, err
	}
	// FIXME: set Att
	msg, err := ret.Receive(0)
	if err == io.EOF {
		return nil, fmt.Errorf("unexpected EOF")
	}
	if msg.Verb == Connect {
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
	if msg.Verb == Error {
		return nil, fmt.Errorf(strings.Join(msg.Args[:1], ""))
	}
	return nil, fmt.Errorf("unexpected verb %v", msg.Verb)
}

func (o *Object) SetJson(val interface{}) error {
	txt, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return o.Set(string(txt))
}

func (o *Object) Set(vals ...string) error {
	_, err := o.Send(&Message{Verb: Set, Args: vals})
	return err
}

func (o *Object) Get(key string) (string, error) {
	ret, err := o.Send(&Message{Verb: Get, Args: []string{key}, Ret: RetPipe})
	if err != nil {
		return "", err
	}
	msg, err := ret.Receive(0)
	if err == io.EOF {
		return "", fmt.Errorf("unexpected EOF")
	}
	if msg.Verb == Set {
		if err != nil {
			return "", err
		}
		if len(msg.Args) != 1 {
			return "", fmt.Errorf("protocol error")
		}
		return msg.Args[0], nil
	}
	if msg.Verb == Error {
		return "", fmt.Errorf(strings.Join(msg.Args[:1], ""))
	}
	return "", fmt.Errorf("unexpected verb %v", msg.Verb)
}

func (o *Object) Watch() (Receiver, error) {
	ret, err := o.Send(&Message{Verb: Watch, Ret: RetPipe})
	if err != nil {
		return nil, err
	}
	msg, err := ret.Receive(0)
	if msg.Verb == Ack {
		return ret, nil
	}
	if msg.Verb == Error {
		return nil, fmt.Errorf(strings.Join(msg.Args[:1], ""))
	}
	return nil, fmt.Errorf("unexpected verb %v", msg.Verb)
}

func (o *Object) Start() error {
	ret, err := o.Send(&Message{Verb: Start, Ret: RetPipe})
	msg, err := ret.Receive(0)
	if err == io.EOF {
		return fmt.Errorf("unexpected EOF")
	}
	if msg.Verb == Ack {
		return nil
	}
	if msg.Verb == Error {
		return fmt.Errorf(strings.Join(msg.Args[:1], ""))
	}
	return fmt.Errorf("unexpected verb %v", msg.Verb)
}

func (o *Object) Stop() error {
	ret, err := o.Send(&Message{Verb: Stop, Ret: RetPipe})
	msg, err := ret.Receive(0)
	if err == io.EOF {
		return fmt.Errorf("unexpected EOF")
	}
	if msg.Verb == Ack {
		return nil
	}
	if msg.Verb == Error {
		return fmt.Errorf(strings.Join(msg.Args[:1], ""))
	}
	return fmt.Errorf("unexpected verb %v", msg.Verb)
}
