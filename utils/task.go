package utils

import (
	"github.com/docker/libswarm"

	"fmt"
	"sync"
)

func Task(f func(in libswarm.Receiver, out libswarm.Sender)) libswarm.Sender {
	var running bool
	var l sync.RWMutex
	inR, inW := libswarm.Pipe()
	outR, outW := libswarm.Pipe()
	obj := libswarm.NewServer()
	obj.OnVerb(libswarm.Attach, libswarm.Handler(func(msg *libswarm.Message) error {
		msg.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: inW})
		fmt.Printf("copying task output from %#v to %#v\n", outR, msg.Ret)
		defer fmt.Printf("(DONE) copying task output from %#v to %#v\n", outR, msg.Ret)
		libswarm.Copy(msg.Ret, outR)
		return nil
	}))
	obj.OnVerb(libswarm.Start, libswarm.Handler(func(msg *libswarm.Message) error {
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
		msg.Ret.Send(&libswarm.Message{Verb: libswarm.Ack})
		return nil
	}))
	return obj
}
