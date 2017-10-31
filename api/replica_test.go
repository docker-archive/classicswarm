package api

import (
	// weird imports for testing
	"net/http/httptest"
	"sync/atomic"
	"testing"

	// we need to import context under a different name
	gocontext "context"
	"crypto/tls"
	"net/http"
	"time"
)

// ContextCheckedEnsurer is a wrapper around context that allows the user to
// ensure that the Done method has been called
type ContextCheckedEnsurer struct {
	gocontext.Context
	// checked is the number of times Done has been called
	checked int64
}

// Done increments the "checked" value and then passes on the underlying
// context's Done method
func (c *ContextCheckedEnsurer) Done() <-chan struct{} {
	// atomic addition so we don't race on done. this isn't actually important
	// right now but it's better to do things the right way the first time
	atomic.AddInt64(&(c.checked), 1)
	return c.Context.Done()
}

// Checked returns the number of times Done has been called
func (c *ContextCheckedEnsurer) Checked() int64 {
	return atomic.LoadInt64(&(c.checked))
}

// TestReplicaBlockedPrimary checks that an empty primary causes ServeHTTP to
// block until the primary is no longer empty
func TestReplicaBlockedPrimary(t *testing.T) {
	// create a new replica
	r := NewReplica(http.NotFoundHandler(), &tls.Config{}, "ouraddr")
	hijackedprimary := ""
	// set a fake hijack method. add a sentinal so we can know it's been called
	r.hijack = func(_ *tls.Config, primary string, _ http.ResponseWriter, _ *http.Request) error {
		hijackedprimary = primary
		return nil
	}

	// create a dummy request. we can use a nil body because we never actually
	// read the body in ServeHTTP
	req, err := http.NewRequest("GET", "/whatever", nil)
	if err != nil {
		t.Fatal("error creating an HTTP request")
	}
	rctx, cancel := gocontext.WithCancel(req.Context())
	// defer cancel so that we don't leave a dangling goroutine if the tests
	// run for a long time
	defer cancel()
	ctx := &ContextCheckedEnsurer{rctx, 0}
	// switch the request context for a ContextCheckedEnsurer so we can
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	// SetPrimary has never been called, so ServeHTTP should block.
	// we can pass a nil ResponseWriter because we don't ever write to it
	// finished
	called := make(chan struct{})
	go func() {
		r.ServeHTTP(rec, req)
		close(called)
	}()
	// wait until we've checked the context at least once, which tells us we've
	// entered the wait on ServeHTTP
	for ctx.Checked() == 0 {
		time.Sleep(100 * time.Millisecond)
		// yeild the proc so that ServeHTTP has a chance to run
	}
	// make sure hijack hasn't been called
	select {
	case <-called:
		t.Fatal("ServeHTTP did not block")
	default:
	}

	// update the primary and make sure that it has unblocked ServeHTTP
	primary := "someprimary"
	r.SetPrimary(primary)
	select {
	case <-called:
		if hijackedprimary != primary {
			t.Fatalf("expected primary %q to be used, but got %q", primary, hijackedprimary)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ServeHTTP was still blocked after 10 seconds")
	}
}
