package api

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
)

func newClientAndScheme(tlsConfig *tls.Config) (*http.Client, string) {
	if tlsConfig != nil {
		return &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}, "https"
	}
	return &http.Client{}, "http"
}

func getContainerFromVars(c *context, vars map[string]string) (*cluster.Container, error) {
	if name, ok := vars["name"]; ok {
		if container := c.cluster.Container(name); container != nil {
			return container, nil
		}
		return nil, fmt.Errorf("No such container: %s", name)

	}
	if ID, ok := vars["execid"]; ok {
		for _, container := range c.cluster.Containers() {
			for _, execID := range container.Info.ExecIDs {
				if ID == execID {
					return container, nil
				}
			}
		}
		return nil, fmt.Errorf("Exec %s not found", ID)
	}
	return nil, errors.New("Not found")
}

// from https://github.com/golang/go/blob/master/src/net/http/httputil/reverseproxy.go#L82
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// prevents leak with https
func closeIdleConnections(client *http.Client) {
	if tr, ok := client.Transport.(*http.Transport); ok {
		tr.CloseIdleConnections()
	}
}

func proxy(tlsConfig *tls.Config, addr string, w http.ResponseWriter, r *http.Request) error {
	// Use a new client for each request
	client, scheme := newClientAndScheme(tlsConfig)
	// RequestURI may not be sent to client
	r.RequestURI = ""

	r.URL.Scheme = scheme
	r.URL.Host = addr

	log.WithFields(log.Fields{"method": r.Method, "url": r.URL}).Debug("Proxy request")
	resp, err := client.Do(r)
	if err != nil {
		return err
	}

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(NewWriteFlusher(w), resp.Body)

	// cleanup
	resp.Body.Close()
	closeIdleConnections(client)

	return nil
}

func hijack(tlsConfig *tls.Config, addr string, w http.ResponseWriter, r *http.Request) error {
	if parts := strings.SplitN(addr, "://", 2); len(parts) == 2 {
		addr = parts[1]
	}

	log.WithField("addr", addr).Debug("Proxy hijack request")

	var (
		d   net.Conn
		err error
	)

	if tlsConfig != nil {
		d, err = tls.Dial("tcp", addr, tlsConfig)
	} else {
		d, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return err
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return err
	}
	nc, _, err := hj.Hijack()
	if err != nil {
		return err
	}
	defer nc.Close()
	defer d.Close()

	err = r.Write(d)
	if err != nil {
		return err
	}

	errc := make(chan error, 2)
	cp := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		if conn, ok := dst.(interface {
			CloseWrite() error
		}); ok {
			conn.CloseWrite()
		}
		errc <- err
	}
	go cp(d, nc)
	go cp(nc, d)
	<-errc
	<-errc

	return nil
}
