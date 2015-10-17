package api

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
)

// Emit an HTTP error and log it.
func httpError(w http.ResponseWriter, err string, status int) {
	log.WithField("status", status).Errorf("HTTP error: %v", err)
	http.Error(w, err, status)
}

func sendJSONMessage(w io.Writer, id, status string) {
	message := struct {
		ID       string      `json:"id,omitempty"`
		Status   string      `json:"status,omitempty"`
		Progress interface{} `json:"progressDetail,omitempty"`
	}{
		id,
		status,
		struct{}{}, // this is required by the docker cli to have a proper display
	}
	json.NewEncoder(w).Encode(message)
}

func sendErrorJSONMessage(w io.Writer, errorCode int, errorMessage string) {
	error := struct {
		Code    int    `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}{
		errorCode,
		errorMessage,
	}

	message := struct {
		ErrorMsg string      `json:"error,omitempty"`
		Error    interface{} `json:"errorDetail,omitempty"`
	}{
		errorMessage,
		&error,
	}

	json.NewEncoder(w).Encode(message)
}
func newClientAndScheme(tlsConfig *tls.Config) (*http.Client, string) {
	if tlsConfig != nil {
		return &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}, "https"
	}
	return &http.Client{}, "http"
}

func getContainerFromVars(c *context, vars map[string]string) (string, *cluster.Container, error) {
	if name, ok := vars["name"]; ok {
		if container := c.cluster.Container(name); container != nil {
			return name, container, nil
		}
		return name, nil, fmt.Errorf("No such container: %s", name)
	}
	if ID, ok := vars["execid"]; ok {
		for _, container := range c.cluster.Containers() {
			for _, execID := range container.Info.ExecIDs {
				if ID == execID {
					return "", container, nil
				}
			}
		}
		return "", nil, fmt.Errorf("Exec %s not found", ID)
	}
	return "", nil, errors.New("Not found")
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

func proxyAsync(tlsConfig *tls.Config, addr string, w http.ResponseWriter, r *http.Request, callback func(*http.Response)) error {
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

	if callback != nil {
		callback(resp)
	}

	// cleanup
	resp.Body.Close()
	closeIdleConnections(client)

	return nil
}

func proxy(tlsConfig *tls.Config, addr string, w http.ResponseWriter, r *http.Request) error {
	return proxyAsync(tlsConfig, addr, w, r, nil)
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

	cp := func(dst io.Writer, src io.Reader, chDone chan struct{}) {
		io.Copy(dst, src)
		if conn, ok := dst.(interface {
			CloseWrite() error
		}); ok {
			conn.CloseWrite()
		}
		close(chDone)
	}
	inDone := make(chan struct{})
	outDone := make(chan struct{})
	go cp(d, nc, inDone)
	go cp(nc, d, outDone)

	// 1. When stdin is done, wait for stdout always
	// 2. When stdout is done, close the stream and wait for stdin to finish
	//
	// On 2, stdin copy should return immediately now since the out stream is closed.
	// Note that we probably don't actually even need to wait here.
	//
	// If we don't close the stream when stdout is done, in some cases stdin will hange
	select {
	case <-inDone:
		// wait for out to be done
		<-outDone
	case <-outDone:
		// close the conn and wait for stdin
		nc.Close()
		<-inDone
	}
	return nil
}

func boolValue(r *http.Request, k string) bool {
	s := strings.ToLower(strings.TrimSpace(r.FormValue(k)))
	return !(s == "" || s == "0" || s == "no" || s == "false" || s == "none")
}

func intValueOrZero(r *http.Request, k string) int {
	val, err := strconv.Atoi(r.FormValue(k))
	if err != nil {
		return 0
	}
	return val
}

func int64ValueOrZero(r *http.Request, k string) int64 {
	val, err := strconv.ParseInt(r.FormValue(k), 10, 64)
	if err != nil {
		return 0
	}
	return val
}
