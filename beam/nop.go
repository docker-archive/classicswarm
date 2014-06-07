package beam

import (
	"io"
)

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
