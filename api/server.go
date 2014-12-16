package api

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler"
)

func newUnixListener(addr string, tlsConfig *tls.Config) (net.Listener, error) {
	if err := syscall.Unlink(addr); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// there is no way to specify the unix rights to use when
	// creating the socket with net.Listener, so we use umask
	// to create the file without rights and then we chmod
	// to the desired unix rights. This prevent unwanted
	// connections between the creation and the chmod
	mask := syscall.Umask(0777)
	defer syscall.Umask(mask)

	l, err := newListener("unix", addr, tlsConfig)
	if err != nil {
		return nil, err
	}

	// only usable by the user who started swarm
	if err := os.Chmod(addr, 0600); err != nil {
		return nil, err
	}

	return l, nil
}

func newListener(proto, addr string, tlsConfig *tls.Config) (net.Listener, error) {
	l, err := net.Listen(proto, addr)
	if err != nil {
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
