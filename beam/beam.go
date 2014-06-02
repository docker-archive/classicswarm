package beam

import (
	"errors"
	"os"
)

type Sender interface {
	Send(msg *Message) (Receiver, error)
	Close() error
}

type Receiver interface {
	Receive(mode int) (*Message, error)
}

type Message struct {
	Name string
	Args []string
	Att  *os.File
	Ret  Sender
}

const (
	Ret int = 1 << iota
	// FIXME: use an `Att` flag to auto-close attachments by default
)

type ReceiverFrom interface {
	ReceiveFrom(Receiver) (int, error)
}

type SenderTo interface {
	SendTo(Sender) (int, error)
}

var (
	ErrIncompatibleSender   = errors.New("incompatible sender")
	ErrIncompatibleReceiver = errors.New("incompatible receiver")
)

// RetPipe is a special value for `Message.Ret`.
// When a Message is sent with `Ret=SendPipe`, the transport must
// substitute it with the writing end of a new pipe, and return the
// other end as a return value.
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

func Repeater(payload *Message) Sender {
	return Handler(func(msg *Message) error {
		msg.Ret.Send(payload)
		return nil
	})
}
