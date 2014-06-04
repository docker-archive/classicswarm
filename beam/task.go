package beam

import (
	"fmt"
	"sync"
)

func Task(f func(in Receiver, out Sender)) Sender {
	var running bool
	var l sync.RWMutex
	inR, inW := Pipe()
	outR, outW := Pipe()
	obj := NewServer()
	obj.OnAttach(Handler(func(msg *Message) error {
		msg.Ret.Send(&Message{Name: "ack", Ret: inW})
		fmt.Printf("copying task output from %#v to %#v\n", outR, msg.Ret)
		defer fmt.Printf("(DONE) copying task output from %#v to %#v\n", outR, msg.Ret)
		Copy(msg.Ret, outR)
		return nil
	}))
	obj.OnStart(Handler(func(msg *Message) error {
		l.RLock()
		r := running
		l.RUnlock()
		if r {
			return fmt.Errorf("already running")
		}
		l.Lock()
		go f(inR, outW)
		running = true
		l.Unlock()
		msg.Ret.Send(&Message{Name: "ack"})
		return nil
	}))
	return obj
}
