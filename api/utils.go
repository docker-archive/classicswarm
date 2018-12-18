package api

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/docker/swarm/cluster"
)

var imageOSErrorPattern = regexp.MustCompile(`image operating system "(.+)" cannot be used on this platform`)

// Emit an HTTP error and log it.
func httpError(w http.ResponseWriter, err string, status int) {
	log.WithField("status", status).Errorf("HTTP error: %v", err)
	http.Error(w, err, status)
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

func getContainerFromVars(c *context, vars map[string]string) (string, *cluster.Container, error) {
	if name, ok := vars["name"]; ok {
		if container := c.cluster.Container(name); container != nil {
			if !container.Engine.IsHealthy() {
				return name, container, fmt.Errorf("Container %s running on unhealthy node %s", name, container.Engine.Name)
			}
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

func proxyAsync(engine *cluster.Engine, w http.ResponseWriter, r *http.Request, callback func(*http.Response)) error {
	// RequestURI may not be sent to client
	r.RequestURI = ""

	client, scheme, err := engine.HTTPClientAndScheme()

	if err != nil {
		return err
	}

	r.URL.Scheme = scheme
	r.URL.Host = engine.Addr

	log.WithFields(log.Fields{"method": r.Method, "url": r.URL}).Debug("Proxy request")

	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		var resp *http.Response
		resp, err = client.Do(r)
		if err != nil {
			return
		}
		// cleanup
		defer resp.Body.Close()

		copyHeader(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		io.Copy(NewWriteFlusher(w), resp.Body)

		if callback != nil {
			callback(resp)
		}
	}()

	type requestCanceler interface {
		CancelRequest(*http.Request)
	}

	var closeNotify <-chan bool
	if closeNotifier, ok := w.(http.CloseNotifier); ok {
		closeNotify = closeNotifier.CloseNotify()
	}
	select {
	case <-closeNotify:
		log.WithFields(log.Fields{"method": r.Method, "url": r.URL}).Debug("user connection closed")
		if rc, ok := client.Transport.(requestCanceler); ok {
			rc.CancelRequest(r)
			log.WithFields(log.Fields{"method": r.Method, "url": r.URL}).Debug("request cancelled")
		}
		// wait for request finish
		<-requestDone
		return err
	case <-requestDone:
		return err
	}
}

func proxy(engine *cluster.Engine, w http.ResponseWriter, r *http.Request) error {
	return proxyAsync(engine, w, r, nil)
}

type tlsClientConn struct {
	*tls.Conn
	rawConn net.Conn
}

func (c *tlsClientConn) CloseWrite() error {
	// Go standard tls.Conn doesn't provide the CloseWrite() method so we do it
	// on its underlying connection.
	if cwc, ok := c.rawConn.(interface {
		CloseWrite() error
	}); ok {
		log.Debug("Calling CloseWrite on Hijacked TLS Conn")
		return cwc.CloseWrite()
	}
	return nil
}

// We need to copy Go's implementation of tls.Dial (pkg/cryptor/tls/tls.go) in
// order to return our custom tlsClientCon struct which holds both the tls.Conn
// object _and_ its underlying raw connection. The rationale for this is that
// we need to be able to close the write end of the connection when attaching,
// which tls.Conn does not provide.
func tlsDialWithDialer(dialer *net.Dialer, network, addr string, config *tls.Config) (net.Conn, error) {
	// We want the Timeout and Deadline values from dialer to cover the
	// whole process: TCP connection and TLS handshake. This means that we
	// also need to start our own timers now.
	timeout := dialer.Timeout

	if !dialer.Deadline.IsZero() {
		deadlineTimeout := dialer.Deadline.Sub(time.Now())
		if timeout == 0 || deadlineTimeout < timeout {
			timeout = deadlineTimeout
		}
	}

	var errChannel chan error

	if timeout != 0 {
		errChannel = make(chan error, 2)
		time.AfterFunc(timeout, func() {
			errChannel <- errors.New("")
		})
	}

	rawConn, err := dialer.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	// When we set up a TCP connection for hijack, there could be long periods
	// of inactivity (a long running command with no output) that in certain
	// network setups may cause ECONNTIMEOUT, leaving the client in an unknown
	// state. Setting TCP KeepAlive on the socket connection will prohibit
	// ECONNTIMEOUT unless the socket connection truly is broken
	if tcpConn, ok := rawConn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	colonPos := strings.LastIndex(addr, ":")
	if colonPos == -1 {
		colonPos = len(addr)
	}
	hostname := addr[:colonPos]

	// If no ServerName is set, infer the ServerName
	// from the hostname we're connecting to.
	if config.ServerName == "" {
		// Make a copy to avoid polluting argument or default.
		c := &tls.Config{
			Rand:                     config.Rand,
			Time:                     config.Time,
			Certificates:             config.Certificates,
			NameToCertificate:        config.NameToCertificate,
			GetCertificate:           config.GetCertificate,
			RootCAs:                  config.RootCAs,
			NextProtos:               config.NextProtos,
			ServerName:               config.ServerName,
			ClientAuth:               config.ClientAuth,
			ClientCAs:                config.ClientCAs,
			InsecureSkipVerify:       config.InsecureSkipVerify,
			CipherSuites:             config.CipherSuites,
			PreferServerCipherSuites: config.PreferServerCipherSuites,
			SessionTicketsDisabled:   config.SessionTicketsDisabled,
			SessionTicketKey:         config.SessionTicketKey,
			ClientSessionCache:       config.ClientSessionCache,
			MinVersion:               config.MinVersion,
			MaxVersion:               config.MaxVersion,
			CurvePreferences:         config.CurvePreferences,
		}
		c.ServerName = hostname
		config = c
	}

	conn := tls.Client(rawConn, config)

	if timeout == 0 {
		err = conn.Handshake()
	} else {
		go func() {
			errChannel <- conn.Handshake()
		}()

		err = <-errChannel
	}

	if err != nil {
		rawConn.Close()
		return nil, err
	}

	// This is Docker difference with standard's crypto/tls package: returned a
	// wrapper which holds both the TLS and raw connections.
	return &tlsClientConn{conn, rawConn}, nil
}

func dialHijack(tlsConfig *tls.Config, addr string) (net.Conn, error) {
	if tlsConfig == nil {
		return net.Dial("tcp", addr)
	}
	return tlsDialWithDialer(new(net.Dialer), "tcp", addr, tlsConfig)
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

	d, err = dialHijack(tlsConfig, addr)
	if err != nil {
		return err
	}

	err = r.Write(d)
	if err != nil {
		return err
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		return errors.New("Docker server does not support hijacking")
	}

	nc, bufrw, err := hj.Hijack()
	if err != nil {
		return err
	}

	defer nc.Close()
	defer d.Close()

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

	// This multi-reader will first read however many bytes were buffered
	// at the time of the hijack, before continuing to read directly from
	// the connection.
	bufferedReader := io.LimitReader(bufrw, int64(bufrw.Reader.Buffered()))
	multiReader := io.MultiReader(bufferedReader, nc)
	go cp(d, multiReader, inDone)

	go cp(nc, d, outDone)

	// 1. When stdin is done, wait for stdout always
	// 2. When stdout is done, close the stream and wait for stdin to finish
	//
	// On 2, stdin copy should return immediately now since the out stream is closed.
	// Note that we probably don't actually even need to wait here.
	//
	// If we don't close the stream when stdout is done, in some cases stdin will hang
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

func tagHasDigest(tag string) bool {
	return strings.Contains(tag, ":")
}

// TODO(nishanttotla): There might be a better way to pass a ref string than construct it here
// getImageRef returns a string containing the registry reference given a repo and tag
func getImageRef(repo, tag string) string {
	ref := repo
	if tag != "" {
		if tagHasDigest(tag) {
			ref += "@" + tag
		} else {
			ref += ":" + tag
		}
	}
	return ref
}

// MatchImageOSError matches a daemon error message that states that an image
// cannot be built on a node because the image has a base image that uses the
// wrong operating system. This function pulls out the required OS from the
// error message.
func MatchImageOSError(errMsg string) string {
	results := imageOSErrorPattern.FindStringSubmatch(errMsg)
	if results == nil || len(results) < 2 {
		return ""
	}
	return results[1]
}

// normalizeEvent takes a cluster Event and ensures backward compatibility
// and all the right fields filled up
func normalizeEvent(receivedEvent *cluster.Event) ([]byte, error) {
	// make a local copy of the event
	e := *receivedEvent
	// make a fresh copy of the Actor.Attributes map to prevent a race condition
	e.Actor.Attributes = make(map[string]string)
	for k, v := range receivedEvent.Actor.Attributes {
		e.Actor.Attributes[k] = v
	}

	// remove this hack once 1.10 is broadly adopted
	e.From = e.From + " node:" + e.Engine.Name

	e.Actor.Attributes["node.name"] = e.Engine.Name
	e.Actor.Attributes["node.id"] = e.Engine.ID
	e.Actor.Attributes["node.addr"] = e.Engine.Addr
	e.Actor.Attributes["node.ip"] = e.Engine.IP

	data, err := json.Marshal(&e)
	if err != nil {
		return nil, err
	}

	// remove the node field once 1.10 is broadly adopted & interlock stops relying on it
	node := fmt.Sprintf(",%q:{%q:%q,%q:%q,%q:%q,%q:%q}}",
		"node",
		"Name", e.Engine.Name,
		"Id", e.Engine.ID,
		"Addr", e.Engine.Addr,
		"Ip", e.Engine.IP,
	)

	// insert Node field
	data = data[:len(data)-1]
	data = append(data, []byte(node)...)

	return data, nil
}
