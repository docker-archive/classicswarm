package inmem

import (
	"io"
	"fmt"
)

type ReceiverFrom interface {
	ReceiveFrom(Receiver) (int, error)
}

type SenderTo interface {
	SendTo(Sender) (int, error)
}

func Copy(dst Sender, src Receiver) (int, error) {
	if senderTo, ok := src.(SenderTo); ok {
		return senderTo.SendTo(dst)
	}
	if receiverFrom, ok := dst.(ReceiverFrom); ok {
		return receiverFrom.ReceiveFrom(src)
	}
	var (
		n int
	)
	for {
		msg, r, w, err := src.Receive(R|W)
		if err == io.EOF {
			break
		}
		if r != nil {
			// FIXME: spawn goroutines to shuttle messages for each
			// level of nested sender/receiver.
			r.Close()
			return n, fmt.Errorf("operation not supported")
		}
		if w != nil {
			// FIXME: spawn goroutines to shuttle messages for each
			// level of nested sender/receiver.
			w.Close()
			return n, fmt.Errorf("operation not supported")
		}
		if err != nil {
			return n, err
		}
		if _, _, err := dst.Send(msg, 0); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

