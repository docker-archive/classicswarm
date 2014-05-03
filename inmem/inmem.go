package inmem

import (
	"fmt"
	"io"
	"sync"
)

type Handle interface {
	Send(msg *Message, mode int) (Handle, error)
	Receive(mode int) (*Message, Handle, error)
	CloseWrite() error
}

const (
	R	= 1 << (32 - 1 - iota)
	W
)

type Message struct {
	Target	string
	Name	string
	Args	[]string
	Data	string
}


func Pipe() (Handle, Handle) {
	red := make(chan *pipeMessage)
	black := make(chan *pipeMessage)
	return &PipeHandle{r: red, w: black}, &PipeHandle{r: black, w: red}
}

type PipeHandle struct {
	sync.RWMutex
	r chan *pipeMessage
	w chan *pipeMessage
}

func (h *PipeHandle) Send(msg *Message, mode int) (Handle, error) {
	h.RLock()
	defer h.RUnlock()
	if h.w == nil {
		return nil, fmt.Errorf("closed pipe")
	}
	var (
		rh Handle
		lh Handle
	)
	if mode&(R|W) != 0 {
		rh, lh = Pipe()
		if  mode&W == 0 {
			lh.CloseWrite()
		}
	}
	h.w <-&pipeMessage{msg, rh}
	return lh, nil
}

func (h *PipeHandle) Receive(mode int) (*Message, Handle, error) {
	pmsg, ok := <-h.r
	if !ok {
		return nil, nil, io.EOF
	}
	var handle Handle
	if pmsg.handle != nil && mode&W == 0 {
		pmsg.handle.CloseWrite()
	}
	if mode&(R|W) != 0 {
		handle = pmsg.handle
	}
	return pmsg.payload, handle, nil
}

func (h *PipeHandle) CloseWrite() error {
	h.Lock()
	defer h.Unlock()
	if h.w == nil {
		return fmt.Errorf("already closed")
	}
	close(h.w)
	h.w = nil
	return nil
}

type pipeMessage struct {
	payload *Message
	handle	Handle
}
