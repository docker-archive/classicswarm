package beam

import (
	"io"
	"sync"
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
	rwait sync.Cond
	wwait sync.Cond
	l     sync.Mutex
	rl    sync.Mutex
	wl    sync.Mutex
	rerr  error // if reader closed, error to give writes
	werr  error // if writer closed, error to give reads
	msg   *Message
}

func (p *pipe) psend(msg *Message) error {
	var err error
	// One writer at a time.
	p.wl.Lock()
	defer p.wl.Unlock()

	p.l.Lock()
	defer p.l.Unlock()
	p.msg = msg
	p.rwait.Signal()
	for {
		if p.msg == nil {
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
	p.msg = nil // in case of rerr or werr
	return err
}

func (p *pipe) send(msg *Message) (ret Receiver, err error) {
	// Prepare nested Receiver if requested
	if RetPipe.Equals(msg.Ret) {
		ret, msg.Ret = Pipe()
	}
	err = p.psend(msg)
	return
}

func (p *pipe) preceive() (*Message, error) {
	p.rl.Lock()
	defer p.rl.Unlock()

	p.l.Lock()
	defer p.l.Unlock()
	for {
		if p.rerr != nil {
			return nil, io.ErrClosedPipe
		}
		if p.msg != nil {
			break
		}
		if p.werr != nil {
			return nil, p.werr
		}
		p.rwait.Wait()
	}
	msg := p.msg
	p.msg = nil
	p.wwait.Signal()
	return msg, nil
}

func (p *pipe) receive(mode int) (*Message, error) {
	msg, err := p.preceive()
	if err != nil {
		return nil, err
	}
	if msg.Ret == nil {
		msg.Ret = NopSender{}
	}
	if mode&Ret == 0 {
		msg.Ret.Close()
	}
	return msg, nil
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

func (r *PipeReceiver) Receive(mode int) (*Message, error) {
	return r.p.receive(mode)
}

func (r *PipeReceiver) SendTo(dst Sender) (int, error) {
	var n int
	// If the destination is a PipeSender, we can cheat
	pdst, ok := dst.(*PipeSender)
	if !ok {
		return 0, ErrIncompatibleSender
	}
	for {
		pmsg, err := r.p.preceive()
		if err == io.EOF {
			break
		}
		if err != nil {
			return n, err
		}
		if err := pdst.p.psend(pmsg); err != nil {
			return n, err
		}
	}
	n++
	return n, nil
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

func (w *PipeSender) Send(msg *Message) (Receiver, error) {
	return w.p.send(msg)
}

func (w *PipeSender) ReceiveFrom(src Receiver) (int, error) {
	var n int
	// If the destination is a PipeReceiver, we can cheat
	psrc, ok := src.(*PipeReceiver)
	if !ok {
		return 0, ErrIncompatibleReceiver
	}
	for {
		pmsg, err := psrc.p.preceive()
		if err == io.EOF {
			break
		}
		if err != nil {
			return n, err
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
