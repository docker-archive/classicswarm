package api

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
)

var localRoutes = []string{"/info", "/_ping"}

// ReverseProxy is a Docker reverse proxy.
type ReverseProxy struct {
	api       http.Handler
	tlsConfig *tls.Config
	dest      string
}

// NewReverseProxy creates a new reverse proxy.
func NewReverseProxy(api http.Handler, tlsConfig *tls.Config) *ReverseProxy {
	return &ReverseProxy{
		api:       api,
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
	// Check whether we should handle this request locally.
	for _, route := range localRoutes {
		if strings.HasSuffix(r.URL.Path, route) {
			p.api.ServeHTTP(w, r)
			return
		}
	}

	// Otherwise, forward.
	if p.dest == "" {
		httpError(w, "No cluster leader", http.StatusInternalServerError)
		return
	}

	if err := hijack(p.tlsConfig, p.dest, w, r); err != nil {
		httpError(w, fmt.Sprintf("Unable to reach cluster leader: %v", err), http.StatusInternalServerError)
	}
}
