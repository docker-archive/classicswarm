package utils

import (
	"github.com/docker/libswarm/beam"
)

type Buffer []*beam.Message

func (buf *Buffer) Send(msg *beam.Message) (beam.Receiver, error) {
	(*buf) = append(*buf, msg)
	return beam.NopReceiver{}, nil
}

func (buf *Buffer) Close() error {
	(*buf) = nil
	return nil
}
