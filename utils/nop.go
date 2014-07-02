package utils

import (
	"github.com/docker/libchan"
	"github.com/docker/libswarm"

	"io"
)

type NopSender struct{}

func (s NopSender) Send(msg *libswarm.Message) (libswarm.Receiver, error) {
	return NopReceiver{}, nil
}

func (s NopSender) Close() error {
	return nil
}

func (s NopSender) Unwrap() libchan.Sender {
	return libchan.NopSender{}
}

type NopReceiver struct{}

func (r NopReceiver) Receive(mode int) (*libswarm.Message, error) {
	return nil, io.EOF
}

func (r NopReceiver) Unwrap() libchan.Receiver {
	return libchan.NopReceiver{}
}
