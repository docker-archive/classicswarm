package beam

import (
	"io"
	"sync"
)

func Copy(dst Sender, src Receiver) (int, error) {
	var tasks sync.WaitGroup
	defer tasks.Wait()
	if senderTo, ok := src.(SenderTo); ok {
		if n, err := senderTo.SendTo(dst); err != ErrIncompatibleSender {
			return n, err
		}
	}
	if receiverFrom, ok := dst.(ReceiverFrom); ok {
		if n, err := receiverFrom.ReceiveFrom(src); err != ErrIncompatibleReceiver {
			return n, err
		}
	}
	var (
		n int
	)
	for {
		msg, err := src.Receive(Ret)
		if err == io.EOF {
			return n, nil
		}
		if err != nil {
			return n, err
		}
		if _, err := dst.Send(msg); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
