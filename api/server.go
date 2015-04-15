package api

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
)

// The default port to listen on for incoming connections
const DefaultDockerPort = ":2375"

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
func ListenAndServe(c cluster.Cluster, hosts []string, enableCors bool, tlsConfig *tls.Config, eventsHandler *EventsHandler) error {
	context := &context{
		cluster:       c,
		eventsHandler: eventsHandler,
		tlsConfig:     tlsConfig,
	}
	r := createRouter(context, enableCors)
	chErrors := make(chan error, len(hosts))

	for _, host := range hosts {
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
					Handler: r,
				}
			)

			switch protoAddrParts[0] {
			case "unix":
				l, err = newUnixListener(protoAddrParts[1], tlsConfig)
			case "tcp":
				l, err = newListener("tcp", protoAddrParts[1], tlsConfig)
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

	for i := 0; i < len(hosts); i++ {
		err := <-chErrors
		if err != nil {
			return err
		}
	}
	return nil
}
