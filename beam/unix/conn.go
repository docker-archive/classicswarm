package unix

import (
	"fmt"
	"os"

	"github.com/docker/libswarm/beam"
	"github.com/docker/libswarm/beam/data"
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

func (c *Conn) Send(msg *beam.Message) (beam.Receiver, error) {
	if msg.Att != nil {
		return nil, fmt.Errorf("file attachment not yet implemented in unix transport")
	}
	parts := []string{msg.Name}
	parts = append(parts, msg.Args...)
	b := []byte(data.EncodeList(parts))
	// Setup nested streams
	var (
		fd  *os.File
		ret beam.Receiver
		err error
	)
	// Caller requested a return pipe
	if beam.RetPipe.Equals(msg.Ret) {
		local, remote, err := sendablePair()
		if err != nil {
			return nil, err
		}
		fd = remote
		ret = &Conn{local}
		// Caller specified its own return channel
	} else if msg.Ret != nil {
		// The specified return channel is a unix conn: engaging cheat mode!
		if retConn, ok := msg.Ret.(*Conn); ok {
			fd, err = retConn.UnixConn.File()
			if err != nil {
				return nil, fmt.Errorf("error passing return channel: %v", err)
			}
			// Close duplicate fd
			retConn.UnixConn.Close()
			// The specified return channel is an unknown type: proxy messages.
		} else {
			local, remote, err := sendablePair()
			if err != nil {
				return nil, fmt.Errorf("error passing return channel: %v", err)
			}
			fd = remote
			// FIXME: do we need a reference no all these background tasks?
			go func() {
				// Copy messages from the remote return channel to the local return channel.
				// When the remote return channel is closed, also close the local return channel.
				localConn := &Conn{local}
				beam.Copy(msg.Ret, localConn)
				msg.Ret.Close()
				localConn.Close()
			}()
		}
	}
	if err := c.UnixConn.Send(b, fd); err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *Conn) Receive(mode int) (*beam.Message, error) {
	b, fd, err := c.UnixConn.Receive()
	if err != nil {
		return nil, err
	}
	parts, n, err := data.DecodeList(string(b))
	if err != nil {
		return nil, err
	}
	if n != len(b) {
		return nil, fmt.Errorf("garbage data %#v", b[:n])
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("malformed message")
	}
	msg := &beam.Message{Name: parts[0], Args: parts[1:]}

	// Apply mode mask
	if fd != nil {
		subconn, err := FileConn(fd)
		if err != nil {
			return nil, err
		}
		fd.Close()
		if mode&beam.Ret != 0 {
			msg.Ret = &Conn{subconn}
		} else {
			subconn.CloseWrite()
		}
	}
	return msg, nil
}
