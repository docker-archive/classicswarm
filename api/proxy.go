package api

import (
	"crypto/tls"
	"net/http"
)

// ReverseProxy is a Docker reverse proxy.
type ReverseProxy struct {
	tlsConfig *tls.Config
	dest      string
}

// NewReverseProxy creates a new reverse proxy.
func NewReverseProxy(tlsConfig *tls.Config) *ReverseProxy {
	return &ReverseProxy{
		tlsConfig: tlsConfig,
	}
}

// SetDestination sets the HTTP destination of the Docker endpoint.
func (p *ReverseProxy) SetDestination(dest string) {
	// FIXME: We have to kill current connections before doing this.
	p.dest = dest
}

// ServeHTTP is the http.Handler.
func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p.dest == "" {
		httpError(w, "No cluster leader", http.StatusInternalServerError)
		return
	}

	if err := hijack(p.tlsConfig, p.dest, w, r); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}
}
