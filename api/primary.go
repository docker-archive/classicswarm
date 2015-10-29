package api

import (
	"crypto/tls"
	"net/http"

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
		"/networks/{networkid:.*}":        proxyNetwork,
		"/volumes":                        getVolumes,
		"/volumes/{volumename:.*}":        proxyVolume,
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
		"/containers/{name:.*}/start":         proxyContainerAndForceRefresh,
		"/containers/{name:.*}/stop":          proxyContainerAndForceRefresh,
		"/containers/{name:.*}/wait":          proxyContainerAndForceRefresh,
		"/containers/{name:.*}/resize":        proxyContainer,
		"/containers/{name:.*}/attach":        proxyHijack,
		"/containers/{name:.*}/copy":          proxyContainer,
		"/containers/{name:.*}/exec":          postContainersExec,
		"/exec/{execid:.*}/start":             postExecStart,
		"/exec/{execid:.*}/resize":            proxyContainer,
		"/networks/create":                    postNetworksCreate,
		"/networks/{networkid:.*}/connect":    proxyNetwork,
		"/networks/{networkid:.*}/disconnect": proxyNetwork,
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
	"OPTIONS": {
		"": optionsHandler,
	},
}

func writeCorsHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Add("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, OPTIONS")
}

// NewPrimary creates a new API router.
func NewPrimary(cluster cluster.Cluster, tlsConfig *tls.Config, status StatusHandler, enableCors bool) *mux.Router {
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
				localFct(context, w, r)
			}
			localMethod := method

			r.Path("/v{version:[0-9.]+}" + localRoute).Methods(localMethod).HandlerFunc(wrap)
			r.Path(localRoute).Methods(localMethod).HandlerFunc(wrap)
		}
	}

	return r
}
