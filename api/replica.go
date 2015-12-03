package api

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
)

var localRoutes = []string{"/info", "/_ping", "/debug"}

// Replica is an API replica that reserves proxy to the primary.
type Replica struct {
	handler   http.Handler
	tlsConfig *tls.Config
	primary   string
}

// NewReplica creates a new API replica.
func NewReplica(handler http.Handler, tlsConfig *tls.Config) *Replica {
	return &Replica{
		handler:   handler,
		tlsConfig: tlsConfig,
	}
}

// SetPrimary sets the address of the primary Swarm manager
func (p *Replica) SetPrimary(primary string) {
	// FIXME: We have to kill current connections before doing this.
	p.primary = primary
}

// ServeHTTP is the http.Handler.
func (p *Replica) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check whether we should handle this request locally.
	for _, route := range localRoutes {
		if strings.HasSuffix(r.URL.Path, route) {
			p.handler.ServeHTTP(w, r)
			return
		}
	}

	// Otherwise, forward.
	if p.primary == "" {
		httpError(w, "No elected primary cluster manager", http.StatusInternalServerError)
		return
	}

	if err := hijack(p.tlsConfig, p.primary, w, r); err != nil {
		httpError(w, fmt.Sprintf("Unable to reach primary cluster manager (%s): %v", err, p.primary), http.StatusInternalServerError)
	}
}
