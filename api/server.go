package api

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler"
)

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

func ListenAndServe(c *cluster.Cluster, s *scheduler.Scheduler, hosts []string, version string, enableCors bool, tlsConfig *tls.Config) error {
	context := &context{
		cluster:       c,
		scheduler:     s,
		version:       version,
		eventsHandler: NewEventsHandler(),
	}
	c.Events(context.eventsHandler)
	r, err := createRouter(context, enableCors)
	if err != nil {
		return err
	}
	chErrors := make(chan error, len(hosts))

	for _, host := range hosts {
		protoAddrParts := strings.SplitN(host, "://", 2)
		if len(protoAddrParts) == 1 {
			protoAddrParts = append([]string{"tcp"}, protoAddrParts...)
		}

		go func() {
			log.Infof("Listening for HTTP on %s (%s)", protoAddrParts[0], protoAddrParts[1])

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
