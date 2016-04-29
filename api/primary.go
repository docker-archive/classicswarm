package api

import (
	"crypto/tls"
	"net/http"

	"net/http/pprof"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/gorilla/mux"
)

// Primary router context, used by handlers.
type context struct {
	cluster       cluster.Cluster
	eventsHandler *eventsHandler
	statusHandler StatusHandler
	debug         bool
	tlsConfig     *tls.Config
	apiVersion    string
}

type handler func(c *context, w http.ResponseWriter, r *http.Request)

var routes = map[string]map[string]handler{
	"HEAD": {
		"/containers/{name:.*}/archive": proxyContainer,
	},
	"GET": {
		"/_ping":                          ping,
		"/events":                         getEvents,
		"/info":                           getInfo,
		"/version":                        getVersion,
		"/images/json":                    getImagesJSON,
		"/images/viz":                     notImplementedHandler,
		"/images/search":                  proxyRandom,
		"/images/get":                     getImages,
		"/images/{name:.*}/get":           proxyImageGet,
		"/images/{name:.*}/history":       proxyImage,
		"/images/{name:.*}/json":          proxyImage,
		"/containers/ps":                  getContainersJSON,
		"/containers/json":                getContainersJSON,
		"/containers/{name:.*}/archive":   proxyContainer,
		"/containers/{name:.*}/export":    proxyContainer,
		"/containers/{name:.*}/changes":   proxyContainer,
		"/containers/{name:.*}/json":      getContainerJSON,
		"/containers/{name:.*}/top":       proxyContainer,
		"/containers/{name:.*}/logs":      proxyContainer,
		"/containers/{name:.*}/stats":     proxyContainer,
		"/containers/{name:.*}/attach/ws": proxyHijack,
		"/exec/{execid:.*}/json":          proxyContainer,
		"/networks":                       getNetworks,
		"/networks/{networkid:.*}":        getNetwork,
		"/volumes":                        getVolumes,
		"/volumes/{volumename:.*}":        getVolume,
	},
	"POST": {
		"/auth":                               proxyRandom,
		"/commit":                             postCommit,
		"/build":                              postBuild,
		"/images/create":                      postImagesCreate,
		"/images/load":                        postImagesLoad,
		"/images/{name:.*}/push":              proxyImagePush,
		"/images/{name:.*}/tag":               postTagImage,
		"/containers/create":                  postContainersCreate,
		"/containers/{name:.*}/kill":          proxyContainerAndForceRefresh,
		"/containers/{name:.*}/pause":         proxyContainerAndForceRefresh,
		"/containers/{name:.*}/unpause":       proxyContainerAndForceRefresh,
		"/containers/{name:.*}/rename":        postRenameContainer,
		"/containers/{name:.*}/restart":       proxyContainerAndForceRefresh,
		"/containers/{name:.*}/start":         postContainersStart,
		"/containers/{name:.*}/stop":          proxyContainerAndForceRefresh,
		"/containers/{name:.*}/update":        proxyContainerAndForceRefresh,
		"/containers/{name:.*}/wait":          proxyContainerAndForceRefresh,
		"/containers/{name:.*}/resize":        proxyContainer,
		"/containers/{name:.*}/attach":        proxyHijack,
		"/containers/{name:.*}/copy":          proxyContainer,
		"/containers/{name:.*}/exec":          postContainersExec,
		"/exec/{execid:.*}/start":             postExecStart,
		"/exec/{execid:.*}/resize":            proxyContainer,
		"/networks/create":                    postNetworksCreate,
		"/networks/{networkid:.*}/connect":    proxyNetworkConnect,
		"/networks/{networkid:.*}/disconnect": proxyNetworkDisconnect,
		"/volumes/create":                     postVolumesCreate,
	},
	"PUT": {
		"/containers/{name:.*}/archive": proxyContainer,
	},
	"DELETE": {
		"/containers/{name:.*}":    deleteContainers,
		"/images/{name:.*}":        deleteImages,
		"/networks/{networkid:.*}": deleteNetworks,
		"/volumes/{name:.*}":       deleteVolumes,
	},
}

func writeCorsHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Add("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, OPTIONS, HEAD")
}

func profilerSetup(mainRouter *mux.Router, path string) {
	var r = mainRouter.PathPrefix(path).Subrouter()
	r.HandleFunc("/pprof/", pprof.Index)
	r.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/pprof/profile", pprof.Profile)
	r.HandleFunc("/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	r.HandleFunc("/pprof/block", pprof.Handler("block").ServeHTTP)
	r.HandleFunc("/pprof/heap", pprof.Handler("heap").ServeHTTP)
	r.HandleFunc("/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	r.HandleFunc("/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
}

// NewPrimary creates a new API router.
func NewPrimary(cluster cluster.Cluster, tlsConfig *tls.Config, status StatusHandler, debug, enableCors bool) *mux.Router {
	// Register the API events handler in the cluster.
	eventsHandler := newEventsHandler()
	cluster.RegisterEventHandler(eventsHandler)

	context := &context{
		cluster:       cluster,
		eventsHandler: eventsHandler,
		statusHandler: status,
		tlsConfig:     tlsConfig,
	}

	r := mux.NewRouter()
	setupPrimaryRouter(r, context, enableCors)

	if debug {
		profilerSetup(r, "/debug/")
	}

	return r
}

func setupPrimaryRouter(r *mux.Router, context *context, enableCors bool) {
	for method, mappings := range routes {
		for route, fct := range mappings {
			log.WithFields(log.Fields{"method": method, "route": route}).Debug("Registering HTTP route")

			localRoute := route
			localFct := fct

			wrap := func(w http.ResponseWriter, r *http.Request) {
				log.WithFields(log.Fields{"method": r.Method, "uri": r.RequestURI}).Debug("HTTP request received")
				if enableCors {
					writeCorsHeaders(w, r)
				}
				context.apiVersion = mux.Vars(r)["version"]
				localFct(context, w, r)
			}
			localMethod := method

			r.Path("/v{version:[0-9]+.[0-9]+}" + localRoute).Methods(localMethod).HandlerFunc(wrap)
			r.Path(localRoute).Methods(localMethod).HandlerFunc(wrap)

			if enableCors {
				optionsMethod := "OPTIONS"
				optionsFct := optionsHandler

				wrap := func(w http.ResponseWriter, r *http.Request) {
					log.WithFields(log.Fields{"method": optionsMethod, "uri": r.RequestURI}).
						Debug("HTTP request received")
					if enableCors {
						writeCorsHeaders(w, r)
					}
					context.apiVersion = mux.Vars(r)["version"]
					optionsFct(context, w, r)
				}

				r.Path("/v{version:[0-9]+.[0-9]+}" + localRoute).
					Methods(optionsMethod).HandlerFunc(wrap)
				r.Path(localRoute).Methods(optionsMethod).
					HandlerFunc(wrap)
			}
		}
	}
}
