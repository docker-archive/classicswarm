package ws

import (
	"errors"
	"github.com/docker/libswarm/beam"
	"github.com/docker/libswarm/beam/http2"
	"github.com/docker/spdystream/ws"
	"github.com/gorilla/websocket"
	"net/http"
)

// Connect to a Beam server over a Websocket connection as a client
func NewSender(wsConn *websocket.Conn) (beam.Sender, error) {
	session, err := http2.NewStreamSession(ws.NewConnection(wsConn))
	if err != nil {
		return nil, err
	}
	return session, nil
}

// Upgrade an HTTP connection to a Beam over HTTP2 over
// Websockets connection.
type Upgrader struct {
	Upgrader websocket.Upgrader
}

func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*http2.Server, error) {
	wsConn, err := u.Upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		return nil, err
	}

	netConn := ws.NewConnection(wsConn)
	server, err := http2.NewServer(netConn)
	if err != nil {
		netConn.Close()
		return nil, err
	}

	return server, nil
}

// Returns true if a handshake error occured in websockets, which means
// a response has already been written to the stream.
func IsHandshakeError(err error) bool {
	_, ok := err.(websocket.HandshakeError)
	return ok
}

type BeamFunc func(beam.Receiver)

// Handler function for serving Beam over HTTP.  Will invoke f and
// then close the server's Beam endpoint after f returns.
func Serve(u *Upgrader, f BeamFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			u.Upgrader.Error(w, r, http.StatusMethodNotAllowed, errors.New("Method not allowed"))
			return
		}

		server, err := u.Upgrade(w, r, nil)
		if err != nil {
			if !IsHandshakeError(err) {
				u.Upgrader.Error(w, r, http.StatusInternalServerError, errors.New("Unable to open an HTTP2 connection over Websockets"))
			}
			return
		}
		defer server.Close()

		f(server)
	}
}
