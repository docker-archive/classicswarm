package inmem

import ()

type Buffer []*Message

func (buf *Buffer) Send(msg *Message, mode int) (Receiver, Sender, error) {
	(*buf) = append(*buf, msg)
	return NopReceiver{}, NopSender{}, nil
}

func (buf *Buffer) Close() error {
	(*buf) = nil
	return nil
}
