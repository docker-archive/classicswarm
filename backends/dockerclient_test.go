package backends

import (
	"github.com/docker/libswarm/beam"

	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type requestStub struct {
	reqMethod string
	reqPath   string
	reqBody   string

	resStatus int
	resBody   string
}

func TestBackendSpawn(t *testing.T) {
	instance(t, nil)
}

func TestAttachAndStart(t *testing.T) {
	i := instance(t, nil)
	_, _, err := i.Attach("")
	if err != nil {
		t.Fatal(err)
	}
	err = i.Start()
	if err != nil {
		t.Fatal(err)
	}
}

func TestLs(t *testing.T) {
	server := makeServer(t, &requestStub{
		reqMethod: "GET",
		reqPath:   "/containers/json",
		resBody: `
      [
        {"Names": ["/foo"]},
        {"Names": ["/bar"]}
      ]
    `,
	})
	i := instance(t, server)
	names, err := i.Ls()
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"foo", "bar"}
	if !reflect.DeepEqual(names, expected) {
		t.Fatalf("expected %#v, got %#v", expected, names)
	}
}

func TestSpawn(t *testing.T) {
	server := makeServer(t, &requestStub{
		reqMethod: "POST",
		reqPath:   "/containers/create",
		reqBody:   "{}",

		resStatus: 201,
		resBody:   "{}",
	})
	i := instance(t, server)
	_, err := i.Spawn("{}")
	if err != nil {
		t.Fatal(err)
	}
}

func TestAttachToChild(t *testing.T) {
	name := "foo"
	server := makeServer(t, &requestStub{
		reqMethod: "GET",
		reqPath:   fmt.Sprintf("/containers/%s/json", name),

		resBody: "{}",
	})
	i := instance(t, server)
	child(t, server, i, name)
}

func TestStartChild(t *testing.T) {
	name := "foo"
	server := makeServer(t, &requestStub{
		reqMethod: "GET",
		reqPath:   fmt.Sprintf("/containers/%s/json", name),

		resBody: "{}",
	}, &requestStub{
		reqMethod: "POST",
		reqPath:   fmt.Sprintf("/containers/%s/start", name),
		reqBody:   "{}",

		resStatus: 204,
	})
	i := instance(t, server)
	c := child(t, server, i, name)
	err := c.Start()
	if err != nil {
		t.Fatal(err)
	}
}

func TestStopChild(t *testing.T) {
	name := "foo"
	server := makeServer(t, &requestStub{
		reqMethod: "GET",
		reqPath:   fmt.Sprintf("/containers/%s/json", name),

		resBody: "{}",
	}, &requestStub{
		reqMethod: "POST",
		reqPath:   fmt.Sprintf("/containers/%s/stop", name),

		resStatus: 204,
	})
	i := instance(t, server)
	c := child(t, server, i, name)
	err := c.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func makeServer(t *testing.T, stubs ...*requestStub) *httptest.Server {
	for _, stub := range stubs {
		stub.reqPath = fmt.Sprintf("/v1.11%s", stub.reqPath)
	}

	stubSummaries := []string{}
	for _, stub := range stubs {
		stubSummaries = append(stubSummaries, fmt.Sprintf("%s %s", stub.reqMethod, stub.reqPath))
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, stub := range stubs {
			if r.Method == stub.reqMethod && r.URL.Path == stub.reqPath {
				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				if string(body) != stub.reqBody {
					t.Fatalf("expected body: %#v, got body: %#v", stub.reqBody, string(body))
				}

				if stub.resStatus > 0 {
					w.WriteHeader(stub.resStatus)
				}
				w.Write([]byte(stub.resBody))
				return
			}
		}

		t.Fatalf("Unexpected request: '%s %s'.\nStubs: %#v", r.Method, r.URL.Path, stubSummaries)
	}))
}

func instance(t *testing.T, server *httptest.Server) *beam.Object {
	url := "tcp://localhost:4243"
	if server != nil {
		url = strings.Replace(server.URL, "http://", "tcp://", 1)
	}

	backend := DockerClient()
	instance, err := beam.Obj(backend).Spawn(url)
	if err != nil {
		t.Fatal(err)
	}
	return instance
}

func child(t *testing.T, server *httptest.Server, i *beam.Object, name string) *beam.Object {
	_, child, err := i.Attach(name)
	if err != nil {
		t.Fatal(err)
	}
	return child
}
