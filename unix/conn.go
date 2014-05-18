package unix

import (
	"fmt"

	"github.com/docker/beam"
	"github.com/docker/beam/data"
)

func Pair() (*Conn, *Conn, error) {
	c1, c2, err := USocketPair()
	if err != nil {
		return nil, nil, err
	}
	return &Conn{c1}, &Conn{c2}, nil
}

type Conn struct {
	*UnixConn
}

func (c *Conn) Send(msg *beam.Message, mode int) (beam.Receiver, beam.Sender, error) {
	if mode != 0 {
		return nil, nil, fmt.Errorf("operation not supported")
	}
	parts := []string{msg.Name}
	parts = append(parts, msg.Args...)
	c.UnixConn.Send([]byte(data.EncodeList(parts)), nil)
	return nil, nil, nil
}

func (c *Conn) Receive(mode int) (*beam.Message, beam.Receiver, beam.Sender, error) {
	if mode != 0 {
		return nil, nil, nil, fmt.Errorf("operation not supported")
	}
	b, _, err := c.UnixConn.Receive()
	if err != nil {
		return nil, nil, nil, err
	}
	parts, n, err := data.DecodeList(string(b))
	if err != nil {
		return nil, nil, nil, err
	}
	if n != len(b) {
		return nil, nil, nil, fmt.Errorf("garbage data %#v", b[:n])
	}
	if len(parts) == 0 {
		return nil, nil, nil, fmt.Errorf("malformed message")
	}
	msg := &beam.Message{parts[0], parts[1:]}
	return msg, nil, nil, nil
}
