package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestRequest(t *testing.T) {
	t.Parallel()

	r := mux.NewRouter()
	setupPrimaryRouter(r, nil, false)
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

	primary := mux.NewRouter()
	setupPrimaryRouter(primary, nil, true)
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
