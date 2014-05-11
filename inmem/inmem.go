package inmem

import (
	"fmt"
	"io"
	"sync"
)

type Sender interface {
	Send(msg *Message, mode int) (Receiver, Sender, error)
}

type Receiver interface {
	Receive(mode int) (*Message, Receiver, Sender, error)
}

type Message struct {
	Name string
	Args []string
	Data []byte
}

const (
	R = 1 << (32 - 1 - iota)
	W
)

func Pipe() (*PipeReceiver, *PipeSender) {
	p := new(pipe)
	p.rwait.L = &p.l
	p.wwait.L = &p.l
	r := &PipeReceiver{p}
	w := &PipeSender{p}
	return r, w
}

type pipe struct {
	ch    chan *pipeMessage
	rwait sync.Cond
	wwait sync.Cond
	l     sync.Mutex
	rl    sync.Mutex
	wl    sync.Mutex
	rerr  error // if reader closed, error to give writes
	werr  error // if writer closed, error to give reads
	pmsg  *pipeMessage
}

type pipeMessage struct {
	msg *Message
	out *PipeSender
	in  *PipeReceiver
}

func (p *pipe) psend(pmsg *pipeMessage) error {
	var err error
	// One writer at a time.
	p.wl.Lock()
	defer p.wl.Unlock()

	p.l.Lock()
	defer p.l.Unlock()
	p.pmsg = pmsg
	p.rwait.Signal()
	for {
		if p.pmsg == nil {
			break
		}
		if p.rerr != nil {
			err = p.rerr
			break
		}
		if p.werr != nil {
			err = io.ErrClosedPipe
		}
		p.wwait.Wait()
	}
	p.pmsg = nil // in case of rerr or werr
	return err
}

func (p *pipe) send(msg *Message, mode int) (in *PipeReceiver, out *PipeSender, err error) {
	// Prepare the message
	pmsg := &pipeMessage{msg: msg}
	if mode&R != 0 {
		in, pmsg.out = Pipe()
		defer func() {
			if err != nil {
				in.Close()
				in = nil
				pmsg.out.Close()
			}
		}()
	}
	if mode&W != 0 {
		pmsg.in, out = Pipe()
		defer func() {
			if err != nil {
				out.Close()
				out = nil
				pmsg.in.Close()
			}
		}()
	}
	err = p.psend(pmsg)
	return
}

func (p *pipe) receive(mode int) (msg *Message, in *PipeReceiver, out *PipeSender, err error) {
	p.rl.Lock()
	defer p.rl.Unlock()

	p.l.Lock()
	defer p.l.Unlock()
	for {
		if p.rerr != nil {
			return nil, nil, nil, io.ErrClosedPipe
		}
		if p.pmsg != nil {
			break
		}
		if p.werr != nil {
			return nil, nil, nil, p.werr
		}
		p.rwait.Wait()
	}
	pmsg := p.pmsg
	if pmsg.out != nil && mode&W == 0 {
		pmsg.out.Close()
		pmsg.out = nil
	}
	if pmsg.in != nil && mode&R == 0 {
		pmsg.in.Close()
		pmsg.in = nil
	}
	p.pmsg = nil
	msg = pmsg.msg
	in = pmsg.in
	out = pmsg.out
	p.wwait.Signal()
	return
}

func (p *pipe) rclose(err error) {
	if err == nil {
		err = io.ErrClosedPipe
	}
	p.l.Lock()
	defer p.l.Unlock()
	p.rerr = err
	p.rwait.Signal()
	p.wwait.Signal()
}

func (p *pipe) wclose(err error) {
	if err == nil {
		err = io.EOF
	}
	p.l.Lock()
	defer p.l.Unlock()
	p.werr = err
	p.rwait.Signal()
	p.wwait.Signal()
}

// PipeReceiver

type PipeReceiver struct {
	p *pipe
}

func (r *PipeReceiver) Receive(mode int) (*Message, Receiver, Sender, error) {
	return r.p.receive(mode)
}

func (r *PipeReceiver) Close() error {
	return r.CloseWithError(nil)
}

func (r *PipeReceiver) CloseWithError(err error) error {
	r.p.rclose(err)
	return nil
}

// PipeSender

type PipeSender struct {
	p *pipe
}

func (w *PipeSender) Send(msg *Message, mode int) (Receiver, Sender, error) {
	return w.p.send(msg, mode)
}

func (w *PipeSender) ReceiveFrom(src Receiver) (int, error) {
	var n int
	for {
		msg, msgr, msgw, err := src.Receive(R | W)
		if err == io.EOF {
			break
		}
		if err != nil {
			return n, err
		}
		pmsg := &pipeMessage{msg: msg}
		if msgr != nil {
			if pmsgr, ok := msgr.(*PipeReceiver); ok {
				pmsg.in = pmsgr
			} else {
				// FIXME: if we're not receiving from a PipeReceiver,
				// we need to create a new pipe and shuttle messages through it
				// in a new goroutine.
				return n, fmt.Errorf("operation not supported")
			}
		}
		if msgw != nil {
			if pmsgw, ok := msgw.(*PipeSender); ok {
				pmsg.out = pmsgw
			} else {
				// FIXME: if we're not sending to a PipeSender,
				// we need to create a new pipe and shuttle messages through it
				// in a new goroutine.
				return n, fmt.Errorf("operation not supported")
			}
		}
		if err := w.p.psend(pmsg); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (w *PipeSender) Close() error {
	return w.CloseWithError(nil)
}

func (w *PipeSender) CloseWithError(err error) error {
	w.p.wclose(err)
	return nil
}
