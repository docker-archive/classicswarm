package unix

import (
	"fmt"
	"os"

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

func sendablePair() (conn *UnixConn, remoteFd *os.File, err error) {
	// Get 2 *os.File
	local, remote, err := SocketPair()
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil {
			local.Close()
			remote.Close()
		}
	}()
	// Convert 1 to *net.UnixConn
	conn, err = FileConn(local)
	if err != nil {
		return nil, nil, err
	}
	local.Close()
	// Return the "mismatched" pair
	return conn, remote, nil
}

// This implements beam.Sender.Close which *only closes the sender*.
// This is similar to the pattern of only closing go channels from
// the sender's side.
// If you want to close the entire connection, call Conn.UnixConn.Close.
func (c *Conn) Close() error {
	return c.UnixConn.CloseWrite()
}

func (c *Conn) Send(msg *beam.Message, mode int) (beam.Receiver, beam.Sender, error) {
	if msg.Att != nil {
		return nil, nil, fmt.Errorf("file attachment not yet implemented in unix transport")
	}
	parts := []string{msg.Name}
	parts = append(parts, msg.Args...)
	b := []byte(data.EncodeList(parts))
	// Setup nested streams
	var (
		fd *os.File
		r  beam.Receiver
		w  beam.Sender
	)
	if mode&(beam.R|beam.W) != 0 {
		local, remote, err := sendablePair()
		if err != nil {
			return nil, nil, err
		}
		fd = remote
		if mode&beam.R != 0 {
			r = &Conn{local}
		}
		if mode&beam.W != 0 {
			w = &Conn{local}
		} else {
			local.CloseWrite()
		}
	}
	c.UnixConn.Send(b, fd)
	return r, w, nil
}

func (c *Conn) Receive(mode int) (*beam.Message, beam.Receiver, beam.Sender, error) {
	b, fd, err := c.UnixConn.Receive()
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
	msg := &beam.Message{Name: parts[0], Args: parts[1:]}

	// Setup nested streams
	var (
		r beam.Receiver
		w beam.Sender
	)
	// Apply mode mask
	if fd != nil {
		subconn, err := FileConn(fd)
		if err != nil {
			return nil, nil, nil, err
		}
		fd.Close()
		if mode&beam.R != 0 {
			r = &Conn{subconn}
		}
		if mode&beam.W != 0 {
			w = &Conn{subconn}
		} else {
			subconn.CloseWrite()
		}
	}
	return msg, r, w, nil
}
