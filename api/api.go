package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type HttpApiFunc func(w http.ResponseWriter, r *http.Request)

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{'O', 'K'})
}

func notImplementedHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not supported in clustering mode.", http.StatusNotImplemented)
}

func createRouter() (*mux.Router, error) {
	r := mux.NewRouter()
	m := map[string]map[string]HttpApiFunc{
		"GET": {
			"/_ping":                          ping,
			"/events":                         notImplementedHandler,
			"/info":                           notImplementedHandler,
			"/version":                        notImplementedHandler,
			"/images/json":                    notImplementedHandler,
			"/images/viz":                     notImplementedHandler,
			"/images/search":                  notImplementedHandler,
			"/images/get":                     notImplementedHandler,
			"/images/{name:.*}/get":           notImplementedHandler,
			"/images/{name:.*}/history":       notImplementedHandler,
			"/images/{name:.*}/json":          notImplementedHandler,
			"/containers/ps":                  notImplementedHandler,
			"/containers/json":                notImplementedHandler,
			"/containers/{name:.*}/export":    notImplementedHandler,
			"/containers/{name:.*}/changes":   notImplementedHandler,
			"/containers/{name:.*}/json":      notImplementedHandler,
			"/containers/{name:.*}/top":       notImplementedHandler,
			"/containers/{name:.*}/logs":      notImplementedHandler,
			"/containers/{name:.*}/attach/ws": notImplementedHandler,
		},
		"POST": {
			"/auth":                         notImplementedHandler,
			"/commit":                       notImplementedHandler,
			"/build":                        notImplementedHandler,
			"/images/create":                notImplementedHandler,
			"/images/load":                  notImplementedHandler,
			"/images/{name:.*}/push":        notImplementedHandler,
			"/images/{name:.*}/tag":         notImplementedHandler,
			"/containers/create":            notImplementedHandler,
			"/containers/{name:.*}/kill":    notImplementedHandler,
			"/containers/{name:.*}/pause":   notImplementedHandler,
			"/containers/{name:.*}/unpause": notImplementedHandler,
			"/containers/{name:.*}/restart": notImplementedHandler,
			"/containers/{name:.*}/start":   notImplementedHandler,
			"/containers/{name:.*}/stop":    notImplementedHandler,
			"/containers/{name:.*}/wait":    notImplementedHandler,
			"/containers/{name:.*}/resize":  notImplementedHandler,
			"/containers/{name:.*}/attach":  notImplementedHandler,
			"/containers/{name:.*}/copy":    notImplementedHandler,
			"/containers/{name:.*}/exec":    notImplementedHandler,
			"/exec/{name:.*}/start":         notImplementedHandler,
			"/exec/{name:.*}/resize":        notImplementedHandler,
		},
		"DELETE": {
			"/containers/{name:.*}": notImplementedHandler,
			"/images/{name:.*}":     notImplementedHandler,
		},
		"OPTIONS": {
			"": notImplementedHandler,
		},
	}

	for method, routes := range m {
		for route, fct := range routes {
			log.Printf("Registering %s, %s", method, route)
			// NOTE: scope issue, make sure the variables are local and won't be changed
			localRoute := route
			localFct := fct
			wrap := func(w http.ResponseWriter, r *http.Request) {
				fmt.Printf("-> %s %s\n", r.Method, r.RequestURI)
				localFct(w, r)
			}
			localMethod := method

			// add the new route
			r.Path("/v{version:[0-9.]+}" + localRoute).Methods(localMethod).HandlerFunc(wrap)
			r.Path(localRoute).Methods(localMethod).HandlerFunc(wrap)
		}
	}

	return r, nil
}

func ListenAndServe(addr string) error {
	r, err := createRouter()
	if err != nil {
		return err
	}
	s := &http.Server{
		Addr:    addr,
		Handler: r,
	}
	return s.ListenAndServe()
}
