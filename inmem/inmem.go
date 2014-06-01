package inmem

import (
	"github.com/docker/beam"
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
	msg *beam.Message
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

func (p *pipe) send(msg *beam.Message, mode int) (in *PipeReceiver, out *PipeSender, err error) {
	// Prepare the message
	pmsg := &pipeMessage{msg: msg}
	if mode&beam.R != 0 {
		in, pmsg.out = Pipe()
		defer func() {
			if err != nil {
				in.Close()
				in = nil
				pmsg.out.Close()
			}
		}()
	}
	if mode&beam.W != 0 {
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

func (p *pipe) preceive() (*pipeMessage, error) {
	p.rl.Lock()
	defer p.rl.Unlock()

	p.l.Lock()
	defer p.l.Unlock()
	for {
		if p.rerr != nil {
			return nil, io.ErrClosedPipe
		}
		if p.pmsg != nil {
			break
		}
		if p.werr != nil {
			return nil, p.werr
		}
		p.rwait.Wait()
	}
	pmsg := p.pmsg
	p.pmsg = nil
	p.wwait.Signal()
	return pmsg, nil
}

func (p *pipe) receive(mode int) (*beam.Message, *PipeReceiver, *PipeSender, error) {
	pmsg, err := p.preceive()
	if err != nil {
		return nil, nil, nil, err
	}
	if pmsg.out != nil && mode&beam.W == 0 {
		pmsg.out.Close()
		pmsg.out = nil
	}
	if pmsg.in != nil && mode&beam.R == 0 {
		pmsg.in.Close()
		pmsg.in = nil
	}
	return pmsg.msg, pmsg.in, pmsg.out, nil
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

func (r *PipeReceiver) Receive(mode int) (*beam.Message, beam.Receiver, beam.Sender, error) {
	msg, pin, pout, err := r.p.receive(mode)
	if err != nil {
		return nil, nil, nil, err
	}
	var (
		// Always return NopReceiver/NopSender instead of nil values,
		// because:
		// - if they were requested in the mode, they can safely be used
		// - if they were not requested, they can safely be ignored (ie no leak if they
		// aren't closed)
		in  beam.Receiver = beam.NopReceiver{}
		out beam.Sender   = beam.NopSender{}
	)
	if pin != nil {
		in = pin
	}
	if pout != nil {
		out = pout
	}
	return msg, in, out, err
}

func (r *PipeReceiver) SendTo(dst beam.Sender) (int, error) {
	var n int
	// If the destination is a PipeSender, we can cheat
	pdst, ok := dst.(*PipeSender)
	if !ok {
		return 0, beam.ErrIncompatibleSender
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

func (w *PipeSender) Send(msg *beam.Message, mode int) (beam.Receiver, beam.Sender, error) {
	pin, pout, err := w.p.send(msg, mode)
	var (
		in  beam.Receiver
		out beam.Sender
	)
	if pin != nil {
		in = pin
	}
	if pout != nil {
		out = pout
	}
	return in, out, err
}

func (w *PipeSender) ReceiveFrom(src beam.Receiver) (int, error) {
	var n int
	// If the destination is a PipeReceiver, we can cheat
	psrc, ok := src.(*PipeReceiver)
	if !ok {
		return 0, beam.ErrIncompatibleReceiver
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
