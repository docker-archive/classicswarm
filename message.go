package libswarm

import (
	"github.com/docker/libchan"
	"github.com/docker/libchan/data"

	"fmt"
	"os"
)

type Message struct {
	Verb
	Args []string
	Ret  Sender
	Att  *os.File
}

type Sender interface {
	Send(msg *Message) (Receiver, error)
	Close() error
	Unwrap() libchan.Sender
}

type Receiver interface {
	Receive(mode int) (*Message, error)
	Unwrap() libchan.Receiver
}

type senderWrapper struct {
	libchan.Sender
}

func WrapSender(s libchan.Sender) Sender {
	return &senderWrapper{s}
}

func (s *senderWrapper) Send(msg *Message) (Receiver, error) {
	recv, err := s.Sender.Send(msg.LibchanMessage())
	if err != nil {
		return nil, err
	}
	return WrapReceiver(recv), err
}

func (s *senderWrapper) Unwrap() libchan.Sender {
	return s.Sender
}

type receiverWrapper struct {
	libchan.Receiver
}

func WrapReceiver(r libchan.Receiver) Receiver {
	return &receiverWrapper{r}
}

func (r *receiverWrapper) Receive(mode int) (*Message, error) {
	lcm, err := r.Receiver.Receive(mode)
	if err != nil {
		return nil, err
	}
	return DecodeLibchanMessage(lcm)
}

func (r *receiverWrapper) Unwrap() libchan.Receiver {
	return r.Receiver
}

type senderUnwrapper struct {
	Sender
}

func (su *senderUnwrapper) Send(lcm *libchan.Message) (libchan.Receiver, error) {
	msg, err := DecodeLibchanMessage(lcm)
	if err != nil {
		return nil, err
	}
	recv, err := su.Sender.Send(msg)
	if err != nil {
		return nil, err
	}
	return &receiverUnwrapper{recv}, nil
}

type receiverUnwrapper struct {
	Receiver
}

func (ru *receiverUnwrapper) Receive(mode int) (*libchan.Message, error) {
	msg, err := ru.Receiver.Receive(mode)
	if err != nil {
		return nil, err
	}
	return msg.LibchanMessage(), nil
}

func Pipe() (Receiver, Sender) {
	r, s := libchan.Pipe()
	return WrapReceiver(r), WrapSender(s)
}

func Copy(s Sender, r Receiver) (int, error) {
	return libchan.Copy(s.Unwrap(), r.Unwrap())
}

func Handler(h func(msg *Message) error) Sender {
	lch := libchan.Handler(func(lcm *libchan.Message) {
		ret := WrapSender(lcm.Ret)
		msg, err := DecodeLibchanMessage(lcm)
		if err != nil {
			ret.Send(&Message{Verb: Error, Args: []string{err.Error()}})
		}
		if err = h(msg); err != nil {
			ret.Send(&Message{Verb: Error, Args: []string{err.Error()}})
		}
	})
	return WrapSender(lch)
}

var RetPipe = WrapSender(libchan.RetPipe)
var Ret = libchan.Ret

var notImplementedMsg = &Message{Verb: Error, Args: []string{"not implemented"}}
var NotImplemented = WrapSender(libchan.Repeater(notImplementedMsg.LibchanMessage()))

func DecodeLibchanMessage(lcm *libchan.Message) (*Message, error) {
	decoded, err := data.Decode(string(lcm.Data))
	if err != nil {
		return nil, err
	}
	verbList, exists := decoded["verb"]
	if !exists {
		return nil, fmt.Errorf("No 'verb' key found in message data: %s", lcm.Data)
	}
	if len(verbList) != 1 {
		return nil, fmt.Errorf("Expected exactly one verb, got %d: %#v", len(verbList), verbList)
	}
	verb, err := VerbFromString(verbList[0])
	if err != nil {
		return nil, err
	}
	args, exists := decoded["args"]
	if !exists {
		return nil, fmt.Errorf("No 'args' key found in message data: %s", lcm.Data)
	}
	return &Message{
		Verb: verb,
		Args: args,
		Ret:  WrapSender(lcm.Ret),
		Att:  lcm.Fd,
	}, nil
}

func (m *Message) LibchanMessage() *libchan.Message {
	encoded := data.Empty().
		Set("verb", m.Verb.String()).
		Set("args", m.Args...)

	var ret libchan.Sender
	if m.Ret != nil {
		ret = m.Ret.Unwrap()
	}

	return &libchan.Message{
		Data: []byte(encoded),
		Ret:  ret,
		Fd:   m.Att,
	}
}
