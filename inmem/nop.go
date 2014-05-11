package inmem

import (
	"io"
)

type NopSender struct{}

func (s NopSender) Send(msg *Message, mode int) (Receiver, Sender, error) {
	return NopReceiver{}, NopSender{}, nil
}

func (s NopSender) Close() error {
	return nil
}

type NopReceiver struct{}

func (r NopReceiver) Receive(mode int) (*Message, Receiver, Sender, error) {
	return nil, nil, nil, io.EOF
}
