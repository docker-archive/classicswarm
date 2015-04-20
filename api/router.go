package api

import (
	"crypto/tls"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/gorilla/mux"
)

// Router context, used by handlers.
type context struct {
	cluster       cluster.Cluster
	eventsHandler *eventsHandler
	debug         bool
	tlsConfig     *tls.Config
}

type handler func(c *context, w http.ResponseWriter, r *http.Request)

var routes = map[string]map[string]handler{
	"GET": {
		"/_ping":                          ping,
		"/events":                         getEvents,
		"/info":                           getInfo,
		"/version":                        getVersion,
		"/images/json":                    getImagesJSON,
		"/images/viz":                     notImplementedHandler,
		"/images/search":                  proxyRandom,
		"/images/get":                     notImplementedHandler,
		"/images/{name:.*}/get":           proxyImage,
		"/images/{name:.*}/history":       proxyImage,
		"/images/{name:.*}/json":          proxyImage,
		"/containers/ps":                  getContainersJSON,
		"/containers/json":                getContainersJSON,
		"/containers/{name:.*}/export":    proxyContainer,
		"/containers/{name:.*}/changes":   proxyContainer,
		"/containers/{name:.*}/json":      getContainerJSON,
		"/containers/{name:.*}/top":       proxyContainer,
		"/containers/{name:.*}/logs":      proxyContainer,
		"/containers/{name:.*}/stats":     proxyContainer,
		"/containers/{name:.*}/attach/ws": proxyHijack,
		"/exec/{execid:.*}/json":          proxyContainer,
	},
	"POST": {
		"/auth":                         proxyRandom,
		"/commit":                       postCommit,
		"/build":                        proxyRandomAndForceRefresh,
		"/images/create":                postImagesCreate,
		"/images/load":                  notImplementedHandler,
		"/images/{name:.*}/push":        proxyImage,
		"/images/{name:.*}/tag":         proxyImageAndForceRefresh,
		"/containers/create":            postContainersCreate,
		"/containers/{name:.*}/kill":    proxyContainer,
		"/containers/{name:.*}/pause":   proxyContainer,
		"/containers/{name:.*}/unpause": proxyContainer,
		"/containers/{name:.*}/rename":  proxyContainer,
		"/containers/{name:.*}/restart": proxyContainer,
		"/containers/{name:.*}/start":   proxyContainer,
		"/containers/{name:.*}/stop":    proxyContainer,
		"/containers/{name:.*}/wait":    proxyContainer,
		"/containers/{name:.*}/resize":  proxyContainer,
		"/containers/{name:.*}/attach":  proxyHijack,
		"/containers/{name:.*}/copy":    proxyContainer,
		"/containers/{name:.*}/exec":    postContainersExec,
		"/exec/{execid:.*}/start":       proxyHijack,
		"/exec/{execid:.*}/resize":      proxyContainer,
	},
	"DELETE": {
		"/containers/{name:.*}": deleteContainers,
		"/images/{name:.*}":     deleteImages,
	},
	"OPTIONS": {
		"": optionsHandler,
	},
}

func writeCorsHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Add("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, OPTIONS")
}

func createRouter(c *context, enableCors bool) *mux.Router {
	r := mux.NewRouter()
	for method, mappings := range routes {
		for route, fct := range mappings {
			log.WithFields(log.Fields{"method": method, "route": route}).Debug("Registering HTTP route")

			localRoute := route
			localFct := fct
			wrap := func(w http.ResponseWriter, r *http.Request) {
				log.WithFields(log.Fields{"method": r.Method, "uri": r.RequestURI}).Info("HTTP request received")
				if enableCors {
					writeCorsHeaders(w, r)
				}
				localFct(c, w, r)
			}
			localMethod := method

			r.Path("/v{version:[0-9.]+}" + localRoute).Methods(localMethod).HandlerFunc(wrap)
			r.Path(localRoute).Methods(localMethod).HandlerFunc(wrap)
		}
	}

	return r
}
