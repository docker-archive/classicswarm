package libswarm

import (
	"github.com/docker/libchan"

	"fmt"
	"io"
)

type Message struct {
	Verb
	Args []string
	Ret  Sender
	Att  io.ReadWriteCloser
}

type Sender interface {
	Send(msg *Message) (Receiver, error)
	Close() error
}

type Receiver interface {
	Receive(mode int) (*Message, error)
}

type internalMessage struct {
	Verb
	Args []string
	Ret  libchan.Sender
	Att  io.ReadWriteCloser
}

type senderWrapper struct {
	libchan.Sender
}

func WrapSender(s libchan.Sender) Sender {
	return &senderWrapper{s}
}

func (s *senderWrapper) Send(msg *Message) (Receiver, error) {
	var rcvr Receiver = NopReceiver{}

	imsg := &internalMessage{
		Verb: msg.Verb,
		Args: msg.Args,
	}

	if msg.Ret != nil {
		thisEnd, otherEnd := libchan.Pipe()

		imsg.Ret = otherEnd

		if RetPipe.Equals(msg.Ret) {
			rcvr = &receiverWrapper{thisEnd}
		} else {
			go Copy(msg.Ret, &receiverWrapper{thisEnd})
		}
	}

	imsg.Att = msg.Att

	return rcvr, s.Sender.Send(imsg)
}

type receiverWrapper struct {
	libchan.Receiver
}

func WrapReceiver(r libchan.Receiver) Receiver {
	return &receiverWrapper{r}
}

func (r *receiverWrapper) Receive(mode int) (*Message, error) {
	imsg := &internalMessage{}
	if err := r.Receiver.Receive(imsg); err != nil {
		return nil, err
	}
	var ret Sender
	if imsg.Ret == nil {
		ret = NopSender{}
	} else {
		ret = &senderWrapper{imsg.Ret}
	}
	if mode&Ret == 0 {
		if err := ret.Close(); err != nil {
			return nil, err
		}
	}
	msg := &Message{
		Verb: imsg.Verb,
		Args: imsg.Args,
		Ret:  ret,
		Att:  imsg.Att,
	}
	return msg, nil
}

func Pipe() (*receiverWrapper, *senderWrapper) {
	r, s := libchan.Pipe()
	return &receiverWrapper{r}, &senderWrapper{s}
}

func Copy(dst Sender, src Receiver) (int, error) {
	var n int
	for {
		msg, err := src.Receive(Ret)
		if err == io.EOF {
			return n, nil
		}
		if err != nil {
			return n, err
		}
		if _, err := dst.Send(msg); err != nil {
			return n, err
		}
		n++
	}
}

type Handler func(msg *Message) error

func (h Handler) Send(msg *Message) (Receiver, error) {
	var ret Receiver
	if RetPipe.Equals(msg.Ret) {
		ret, msg.Ret = Pipe()
	}
	go func() {
		if msg.Ret == nil {
			msg.Ret = NopSender{}
		}
		h(msg)
		msg.Ret.Close()
	}()
	return ret, nil
}

func (h Handler) Close() error {
	return fmt.Errorf("can't close a Handler")
}

func Repeater(payload *Message) Sender {
	return Handler(func(msg *Message) error {
		msg.Ret.Send(payload)
		return nil
	})
}

var notImplementedMsg = &Message{Verb: Error, Args: []string{"not implemented"}}
var NotImplemented = Repeater(notImplementedMsg)

const (
	Ret int = 1 << iota
	// FIXME: use an `Att` flag to auto-close attachments by default
)

type retPipe struct {
	NopSender
}

var RetPipe = retPipe{}

func (r retPipe) Equals(val Sender) bool {
	if rval, ok := val.(retPipe); ok {
		return rval == r
	}
	return false
}

type NopSender struct{}

func (s NopSender) Send(msg *Message) (Receiver, error) {
	return NopReceiver{}, nil
}

func (s NopSender) Close() error {
	return nil
}

type NopReceiver struct{}

func (r NopReceiver) Receive(mode int) (*Message, error) {
	return nil, io.EOF
}
