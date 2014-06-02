package utils

import (
	"github.com/docker/libswarm/beam"
	"sync"
)

func Copy(dst beam.Sender, src beam.Receiver) (int, error) {
	var tasks sync.WaitGroup
	defer tasks.Wait()
	if senderTo, ok := src.(beam.SenderTo); ok {
		if n, err := senderTo.SendTo(dst); err != beam.ErrIncompatibleSender {
			return n, err
		}
	}
	if receiverFrom, ok := dst.(beam.ReceiverFrom); ok {
		if n, err := receiverFrom.ReceiveFrom(src); err != beam.ErrIncompatibleReceiver {
			return n, err
		}
	}
	var (
		n int
	)
	copyAndClose := func(dst beam.Sender, src beam.Receiver) {
		if dst == nil {
			return
		}
		defer dst.Close()
		if src == nil {
			return
		}
		Copy(dst, src)
	}
	for {
		msg, rcvR, rcvW, err := src.Receive(beam.R | beam.W)
		if err != nil {
			return n, err
		}
		sndR, sndW, err := dst.Send(msg, beam.R|beam.W)
		if err != nil {
			if rcvW != nil {
				rcvW.Close()
			}
			return n, err
		}
		tasks.Add(2)
		go func() {
			copyAndClose(rcvW, sndR)
			tasks.Done()
		}()
		go func() {
			copyAndClose(sndW, rcvR)
			tasks.Done()
		}()
		n++
	}
	return n, nil
}
