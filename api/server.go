package api

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

// DefaultDockerPort is the default port to listen on for incoming connections.
const DefaultDockerPort = ":2375"

// Dispatcher is a meta http.Handler. It acts as an http.Handler and forwards
// requests to another http.Handler that can be changed at runtime.
type dispatcher struct {
	handler http.Handler
}

// SetHandler changes the underlying handler.
func (d *dispatcher) SetHandler(handler http.Handler) {
	d.handler = handler
}

// ServeHTTP forwards requests to the underlying handler.
func (d *dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if d.handler == nil {
		httpError(w, "No dispatcher defined", http.StatusInternalServerError)
		return
	}
	d.handler.ServeHTTP(w, r)
}

// Server is a Docker API server.
type Server struct {
	hosts      []string
	tlsConfig  *tls.Config
	dispatcher *dispatcher
}

// NewServer creates an api.Server.
func NewServer(hosts []string, tlsConfig *tls.Config) *Server {
	return &Server{
		hosts:      hosts,
		tlsConfig:  tlsConfig,
		dispatcher: &dispatcher{},
	}
}

// SetHandler is used to overwrite the HTTP handler for the API.
// This can be the api router or a reverse proxy.
func (s *Server) SetHandler(handler http.Handler) {
	s.dispatcher.SetHandler(handler)
}

func newListener(proto, addr string, tlsConfig *tls.Config) (net.Listener, error) {
	l, err := net.Listen(proto, addr)
	if err != nil {
		if strings.Contains(err.Error(), "address already in use") && strings.Contains(addr, DefaultDockerPort) {
			return nil, fmt.Errorf("%s: is Docker already running on this machine? Try using a different port", err)
		}
		return nil, err
	}
	if tlsConfig != nil {
		tlsConfig.NextProtos = []string{"http/1.1"}
		l = tls.NewListener(l, tlsConfig)
	}
	return l, nil
}

// ListenAndServe starts an HTTP server on each host to listen on its
// TCP or Unix network address and calls Serve on each host's server
// to handle requests on incoming connections.
//
// The expected format for a host string is [protocol://]address. The protocol
// must be either "tcp" or "unix", with "tcp" used by default if not specified.
func (s *Server) ListenAndServe() error {
	chErrors := make(chan error, len(s.hosts))

	for _, host := range s.hosts {
		protoAddrParts := strings.SplitN(host, "://", 2)
		if len(protoAddrParts) == 1 {
			protoAddrParts = append([]string{"tcp"}, protoAddrParts...)
		}

		go func() {
			log.WithFields(log.Fields{"proto": protoAddrParts[0], "addr": protoAddrParts[1]}).Info("Listening for HTTP")

			var (
				l      net.Listener
				err    error
				server = &http.Server{
					Addr:    protoAddrParts[1],
					Handler: s.dispatcher,
				}
			)

			switch protoAddrParts[0] {
			case "unix":
				l, err = newUnixListener(protoAddrParts[1], s.tlsConfig)
			case "tcp":
				l, err = newListener("tcp", protoAddrParts[1], s.tlsConfig)
			default:
				err = fmt.Errorf("unsupported protocol: %q", protoAddrParts[0])
			}
			if err != nil {
				chErrors <- err
			} else {
				chErrors <- server.Serve(l)
			}

		}()
	}

	for i := 0; i < len(s.hosts); i++ {
		err := <-chErrors
		if err != nil {
			return err
		}
	}
	return nil
}
