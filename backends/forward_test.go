package backends

import (
	"bytes"
	"fmt"
	"github.com/dotcloud/docker/engine"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContainers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/containers/json" {
			t.Fatalf("expected HTTP request to /containers/json, got %s", r.URL.Path)
		}
		fmt.Fprint(w, "[]")
	}))
	defer ts.Close()

	eng, err := newEngine(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	job := eng.Job("containers")
	stdout := new(bytes.Buffer)
	job.Stdout.Add(stdout)

	err = job.Run()
	if err != nil {
		t.Fatal(err)
	}

	if stdout.String() != "[]" {
		t.Fatalf("expected [], got %#v", stdout.String())
	}
}

func newEngine(url string) (*engine.Engine, error) {
	eng := engine.New()
	err := Forward().Install(eng)
	if err != nil {
		return nil, err
	}
	err = eng.Job("forward", url).Run()
	if err != nil {
		return nil, err
	}
	return eng, nil
}
