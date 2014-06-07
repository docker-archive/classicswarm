package backends

import (
	"encoding/json"
	"fmt"
	"github.com/docker/libswarm/beam"
	"github.com/dotcloud/docker/api"
	"github.com/dotcloud/docker/pkg/version"
	"github.com/gorilla/mux"
	"net"
	"net/http"
)

func ApiServer() beam.Sender {
	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
		instance := beam.Task(func(in beam.Receiver, out beam.Sender) {
			err := listenAndServe("tcp", "0.0.0.0:4243", out)
			if err != nil {
				fmt.Printf("listenAndServe: %v", err)
			}
		})
		_, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: instance})
		return err
	}))
	return backend
}

type HttpApiFunc func(out beam.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error

func listenAndServe(proto, addr string, out beam.Sender) error {
	fmt.Println("Starting API server...")
	r, err := createRouter(out)
	if err != nil {
		return err
	}
	l, err := net.Listen(proto, addr)
	if err != nil {
		return err
	}
	httpSrv := http.Server{Addr: addr, Handler: r}
	return httpSrv.Serve(l)
}

func ping(out beam.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	_, err := w.Write([]byte{'O', 'K'})
	return err
}

func getContainersJSON(out beam.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	o := beam.Obj(out)
	names, err := o.Ls()
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, names)
}

func postContainersStart(out beam.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	// TODO: r.Body

	name := vars["name"]
	_, containerOut, err := beam.Obj(out).Attach(name)
	container := beam.Obj(containerOut)
	if err != nil {
		return err
	}
	if err := container.Start(); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func createRouter(out beam.Sender) (*mux.Router, error) {
	r := mux.NewRouter()
	m := map[string]map[string]HttpApiFunc{
		"GET": {
			"/_ping":           ping,
			"/containers/json": getContainersJSON,
		},
		"POST": {
			"/containers/{name:.*}/start": postContainersStart,
		},
		"DELETE":  {},
		"OPTIONS": {},
	}

	for method, routes := range m {
		for route, fct := range routes {
			localRoute := route
			localFct := fct
			localMethod := method

			f := makeHttpHandler(out, localMethod, localRoute, localFct, version.Version("0.11.0"))

			// add the new route
			if localRoute == "" {
				r.Methods(localMethod).HandlerFunc(f)
			} else {
				r.Path("/v{version:[0-9.]+}" + localRoute).Methods(localMethod).HandlerFunc(f)
				r.Path(localRoute).Methods(localMethod).HandlerFunc(f)
			}
		}
	}

	return r, nil
}

func makeHttpHandler(out beam.Sender, localMethod string, localRoute string, handlerFunc HttpApiFunc, dockerVersion version.Version) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// log the request
		fmt.Printf("Calling %s %s\n", localMethod, localRoute)

		version := version.Version(mux.Vars(r)["version"])
		if version == "" {
			version = api.APIVERSION
		}

		if err := handlerFunc(out, version, w, r, mux.Vars(r)); err != nil {
			fmt.Printf("Error: %s", err)
		}
	}
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	return enc.Encode(v)
}
