package beam

import (
	"fmt"
)

type Handler func(msg *Message) error

func (h Handler) Send(msg *Message) (Receiver, error) {
	var ret Receiver
	if RetPipe.Equals(msg.Ret) {
		ret, msg.Ret = Pipe()
	}
	go func() {
		// Ret must always be a valid Sender, so handlers
		// can safely send to it
		if msg.Ret == nil {
			msg.Ret = NopSender{}
		}
		err := h(msg)
		if err != nil {
			Obj(msg.Ret).Error("%v", err)
		}
		msg.Ret.Close()
	}()
	return ret, nil
}

func (h Handler) Close() error {
	return fmt.Errorf("can't close")
}
