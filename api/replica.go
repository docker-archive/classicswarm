package api

import (
	// we have to import this as `gocontext` because there is already a type
	// called `context` declared in primary.go
	gocontext "context"

	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

var localRoutes = []string{"/_ping", "/info", "/debug"}

// Replica is an API replica that reserves proxy to the primary.
type Replica struct {
	handler   http.Handler
	tlsConfig *tls.Config
	primary   string
	// the address of this replica
	addr string
	// mu is the mutex used for locking the primary.
	mu sync.RWMutex
	// primaryWait is used to block execution of ServeHTTP until the primary is
	// ready.
	primaryWait *sync.Cond

	// capture the hijack function for easier unit testing
	hijack func(*tls.Config, string, http.ResponseWriter, *http.Request) error
}

// NewReplica creates a new API replica.
func NewReplica(handler http.Handler, tlsConfig *tls.Config, addr string) *Replica {
	r := &Replica{
		handler:   handler,
		tlsConfig: tlsConfig,
		addr:      addr,
		hijack:    hijack,
	}
	// This seems to be a big confusing, so here's the explanation:
	// A Cond is used to wait for a certain value to happen. You use it by:
	//
	//   1. Acquiring a lock (cond.L.Lock())
	//   2. Checking the synchronized value
	//   3. Calling cond.Wait() if the value isn't what you want
	//      a. wait releases the lock (cond.L.Unlock()) when it is entered
	//      b. wait reacquires the lock (cond.L.Lock()) when it is left.
	//   4. Checking the value again (it may have changed!)
	//   5. Doing your business
	//   6. Releasing the lock (cond.L.Unlock())
	//
	// The issue here is that for optimal performance, because the ServeHTTP
	// function only reads the primary value, we want to use a RWMutex, so that
	// getPrimary can acquire a RLock and not block other concurrent handlers,
	// but SetPrimary can acquire a write lock `Lock` to modify that value.
	//
	// NewCond takes a `Locker` interface, which implements `Lock` and
	// `Unlock`, and doesn't have any conception of RLock/RUnlock. But we need
	// those methods so that `r.primaryWait.Wait()` releases and reacquires the
	// correct lock, the Read lock.
	//
	// The way we do this is by passing the return of the `RLocker` method.
	// This returned object has `Lock` and `Unlock` methods which coorespond to
	// the `RLock` and `RUnlock` methods of the RWMutex. This means that calls
	// to `Wait` will acquire and release the read lock, not the write lock.
	//
	// This way, `SetPrimary` acquires and releases the WRITE lock on mu
	// (mu.Lock) and `getPrimary` acquires and releases a READ lock on mu
	// (r.primaryWait.L.Lock).
	r.primaryWait = sync.NewCond(r.mu.RLocker())
	return r
}

// SetPrimary sets the address of the primary Swarm manager
func (p *Replica) SetPrimary(primary string) {
	// FIXME: We have to kill current connections before doing this.
	// this lock on the mutex is a write lock
	p.mu.Lock()
	defer p.mu.Unlock()
	p.primary = primary
	if p.primary != "" {
		// broadcast to wake everyone waiting on a nonempty primary
		p.primaryWait.Broadcast()
	}
}

// getPrimary returns the primary if it exists, blocks if it doesn't, and
// returns an error if the context is canceled. It's packed nicely in a method
// to cleanly encapsulate synchronized part of ServeHTTP.
func (p *Replica) getPrimary(ctx gocontext.Context) (string, error) {
	// lock a read on primary.
	//
	// use primaryWait.L just to ensure we're locking and unlocking the correct
	// lock, so if we change the lock type later, we can't accidentally forget
	// to update which lock is acquired and released by the Wait.
	//
	// see the description in NewReplica for why this is a read lock
	p.primaryWait.L.Lock()
	defer p.primaryWait.L.Unlock()
	// check if the primary is empty, and launch into a wait if it is
	for p.primary == "" {
		// give up waiting if the context is canceled, to avoid blocking forever
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("No elected primary cluster manager: %v", ctx.Err())
		default:
		}
		// wait for the primary to become available
		p.primaryWait.Wait()
	}
	return p.primary, nil
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
	primary, err := p.getPrimary(r.Context())
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}

	// if we've become the primary, go ahead and serve the request ourself.
	// this happens if we're waiting for a primary and then we become the
	// primary
	if primary == p.addr {
		p.handler.ServeHTTP(w, r)
		return
	}
	if err := p.hijack(p.tlsConfig, primary, w, r); err != nil {
		httpError(w, fmt.Sprintf("Unable to reach primary cluster manager (%s): %v", err, p.primary), http.StatusInternalServerError)
	}
}
