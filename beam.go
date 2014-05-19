package beam

import ()

type Sender interface {
	Send(msg *Message, mode int) (Receiver, Sender, error)
	Close() error
}

type Receiver interface {
	Receive(mode int) (*Message, Receiver, Sender, error)
}

type Message struct {
	Name string
	Args []string
}

const (
	R = 1 << (32 - 1 - iota)
	W
)
