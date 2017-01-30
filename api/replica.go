package api

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var localRoutes = []string{"/_ping", "/info", "/debug"}

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

func (p *Replica) ping(w http.ResponseWriter) {
	if p.primary == "" {
		httpError(w, "No elected primary cluster manager", http.StatusInternalServerError)
		return
	}
	w.Write([]byte("OK"))
}

// ServeHTTP is the http.Handler.
func (p *Replica) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check whether we should handle this request locally.
	// _ping is handled specially because it needs to check the p.primary
	// field.
	if strings.HasSuffix(r.URL.Path, "/_ping") {
		p.ping(w)
		return
	}
	for _, route := range localRoutes {
		if strings.HasSuffix(r.URL.Path, route) {
			p.handler.ServeHTTP(w, r)
			return
		}
	}

	for i := 0; i < 60; i++ {
		if p.primary != "" || r.Context().Err() != nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
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
