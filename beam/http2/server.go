package http2

import (
	"github.com/docker/libswarm/beam"
	"github.com/docker/spdystream"
	"net"
	"sync"
)

// Serve a Beam endpoint over a single HTTP2 connection
type Server struct {
	conn           *spdystream.Connection
	streamChan     chan *spdystream.Stream
	streamLock     sync.RWMutex
	subStreamChans map[string]chan *spdystream.Stream
}

// Create a Beam receiver from a net.Conn
func NewServer(conn net.Conn) (*Server, error) {
	spdyConn, err := spdystream.NewConnection(conn, true)
	if err != nil {
		return nil, err
	}

	s := &Server{
		conn:           spdyConn,
		streamChan:     make(chan *spdystream.Stream),
		subStreamChans: make(map[string]chan *spdystream.Stream),
	}
	go s.conn.Serve(s.streamHandler, spdystream.NoAuthHandler)

	return s, nil
}

func (s *Server) Close() error {
	return s.conn.Close()
}

func (s *Server) Receive(mode int) (*beam.Message, error) {
	stream := <-s.streamChan
	return createStreamMessage(stream, mode, s, nil)
}

func (s *Server) streamHandler(stream *spdystream.Stream) {
	streamChan := s.getStreamChan(stream.Parent())
	streamChan <- stream
}

func (s *Server) addStreamChan(stream *spdystream.Stream, streamChan chan *spdystream.Stream) {
	s.streamLock.Lock()
	s.subStreamChans[stream.String()] = streamChan
	s.streamLock.Unlock()
}

func (s *Server) getStreamChan(stream *spdystream.Stream) chan *spdystream.Stream {
	if stream == nil {
		return s.streamChan
	}
	s.streamLock.RLock()
	defer s.streamLock.RUnlock()
	streamChan, ok := s.subStreamChans[stream.String()]
	if ok {
		return streamChan
	}
	return s.streamChan
}
