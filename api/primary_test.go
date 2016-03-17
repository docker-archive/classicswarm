package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

	if w.Code != 200 {
		t.Fatalf("HTTP response status code is %d, not 200", w.Code)
	}
}

func TestCorsRequest(t *testing.T) {
	t.Parallel()

	context := &context{}
	primary := mux.NewRouter()
	setupPrimaryRouter(primary, context, true)
	w := httptest.NewRecorder()

	// test an OPTIONS request when cors enabled
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

	// test response status code
	if w.Code != 200 {
		t.Fatalf("HTTP response status code is %d, not 200", w.Code)
	}

	// test Access-Control-Allow-Headers in response headers
	allowHeaders := w.Header().Get("Access-Control-Allow-Headers")

	for _, value := range []string{"Origin", "X-Requested-With", "Content-Type", "Accept", "X-Registry-Auth"} {
		if !strings.Contains(allowHeaders, value) {
			t.Fatalf("Header %s is not in Access-Control-Allow-Headers", value)
		}
	}

	// test Access-Control-Allow-Methods in response headers
	allowMethods := w.Header().Get("Access-Control-Allow-Methods")

	for _, value := range []string{"GET", "POST", "DELETE", "PUT", "OPTIONS", "HEAD"} {
		if !strings.Contains(allowMethods, value) {
			t.Fatalf("Method %s is not in Access-Control-Allow-Methods", value)
		}
	}

	// test a normal request ( GET /_ping ) when cors enabled
	w2 := httptest.NewRecorder()

	r2, e2 := http.NewRequest("GET", "/_ping", nil)
	if nil != e2 {
		t.Fatalf("couldn't set up test request")
	}

	primary.ServeHTTP(w2, r2)

	if w2.Body.String() != "OK" {
		t.Fatalf("couldn't get body content when cors enabled")
	}

	if w2.Code != 200 {
		t.Fatalf("HTTP response status code is %d, not 200", w2.Code)
	}
}
