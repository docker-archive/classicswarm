package inmem

import (
	"github.com/docker/beam"
	"io"
)

type NopSender struct{}

func (s NopSender) Send(msg *beam.Message, mode int) (beam.Receiver, beam.Sender, error) {
	return NopReceiver{}, NopSender{}, nil
}

func (s NopSender) Close() error {
	return nil
}

type NopReceiver struct{}

func (r NopReceiver) Receive(mode int) (*beam.Message, beam.Receiver, beam.Sender, error) {
	return nil, nil, nil, io.EOF
}
