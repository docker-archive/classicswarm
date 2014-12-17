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

func getContainerFromVars(c *context, vars map[string]string) (*cluster.Container, error) {
	if name, ok := vars["name"]; ok {
		if container := c.cluster.Container(name); container != nil {
			return container, nil
		}
		return nil, fmt.Errorf("Container %s not found", name)

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

func proxy(tlsConfig *tls.Config, container *cluster.Container, w http.ResponseWriter, r *http.Request) error {
	// Use a new client for each request
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// RequestURI may not be sent to client
	r.RequestURI = ""

	parts := strings.SplitN(container.Node.Addr, "://", 2)
	if len(parts) == 2 {
		r.URL.Scheme = parts[0]
		r.URL.Host = parts[1]
	} else {
		r.URL.Scheme = "http"
		r.URL.Host = parts[0]
	}

	log.Debugf("[PROXY] --> %s %s", r.Method, r.URL)
	resp, err := client.Do(r)
	if err != nil {
		return err
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	return nil
}

func hijack(tlsConfig *tls.Config, container *cluster.Container, w http.ResponseWriter, r *http.Request) error {
	addr := container.Node.Addr
	if parts := strings.SplitN(container.Node.Addr, "://", 2); len(parts) == 2 {
		addr = parts[1]
	}

	log.Debugf("[HIJACK PROXY] --> %s", addr)

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
