package inmem

import (
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
	return n, fmt.Errorf("operation not supported")
}

