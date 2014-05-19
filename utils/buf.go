package utils

import (
	"github.com/docker/beam"
)

type Buffer []*beam.Message

func (buf *Buffer) Send(msg *beam.Message, mode int) (beam.Receiver, beam.Sender, error) {
	(*buf) = append(*buf, msg)
	return NopReceiver{}, NopSender{}, nil
}

func (buf *Buffer) Close() error {
	(*buf) = nil
	return nil
}
