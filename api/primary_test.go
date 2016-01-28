package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestRequest(t *testing.T) {
	t.Parallel()

	context := &context{}
	r := mux.NewRouter()
	setupPrimaryRouter(r, context, false)
	w := httptest.NewRecorder()

	req, e := http.NewRequest("GET", "/version", nil)
	if nil != e {
		t.Fatalf("couldn't set up test request")
	}

	r.ServeHTTP(w, req)

	t.Logf("%d - %s", w.Code, w.Body.String())
	t.Log("Expecting no extra Headers")
	for k, v := range w.Header() {
		t.Log(k, " : ", v)
	}

	if w.Code == 404 {
		t.Fatalf("failed not found")
	}
}

func TestCorsRequest(t *testing.T) {
	t.Parallel()

	context := &context{}
	primary := mux.NewRouter()
	setupPrimaryRouter(primary, context, true)
	w := httptest.NewRecorder()

	r, e := http.NewRequest("OPTIONS", "/version", nil)
	if nil != e {
		t.Fatalf("couldn't set up test request")
	}

	primary.ServeHTTP(w, r)

	t.Logf("%d - %s", w.Code, w.Body.String())
	t.Log("Expecting extra cors Headers")
	for k, v := range w.Header() {
		t.Log(k, " : ", v)
	}

	if w.Code == 404 {
		t.Fatalf("failed not found")
	}
}
