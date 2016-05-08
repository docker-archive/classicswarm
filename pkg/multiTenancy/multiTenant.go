package multiTenancy

import (
	"net/http"
	"strings"
	"os"
	"container/list"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
//	"github.com/docker/swarm/pkg/authZ/keystone"
	"github.com/docker/swarm/pkg/multiTenancy/handler"
	"github.com/docker/swarm/pkg/authZ/states"
	"github.com/docker/swarm/pkg/authN"
	"github.com/docker/swarm/pkg/authZ"
)

//Hooks - Entry point to AuthZ mechanisem
type MultiTenant struct{}

var plugins=list.New() // plugins list

//Handle - Hook point from primary to plugins
func (*MultiTenant) Handle(cluster cluster.Cluster, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debug("In multiTenant.Handle ...")	
		if (os.Getenv("SWARM_MULTI_TENANT") == "false") {
			next.ServeHTTP(w, r)
			return
		}
		eventType := eventParse(r)
//		aclsAPI.HandleRequest(cluster, eventType, w, r, next)      
//        flist := []handler{new(authN.DefaultAuthNImpl).Handle};
//        err := flist[0](cluster, eventType, w, r, next);

		plugin := plugins.Front()
        log.Debug(plugins.Len())
        plugins.Remove(plugins.Front())
        err := plugin.Value.(handler.Handler)(cluster, eventType, w, r, next, plugins)
        if err != nil {
        	log.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})
}



func eventParse(r *http.Request) states.EventEnum {
	log.Debug("multiTenant.eventParse uri: ", r.RequestURI)

	if strings.Contains(r.RequestURI, "/containers") && (strings.Contains(r.RequestURI, "create")) {
		return states.ContainerCreate
	}

	if strings.Contains(r.RequestURI, "/containers/json") {
		return states.ContainersList
	}

	if strings.Contains(r.RequestURI, "/containers") &&
		(strings.Contains(r.RequestURI, "logs") || strings.Contains(r.RequestURI, "attach") || strings.HasSuffix(r.RequestURI, "exec")) {
		return states.StreamOrHijack
	}
	if strings.Contains(r.RequestURI, "/containers") && strings.HasSuffix(r.RequestURI, "/json") {
		return states.ContainerInspect
	}
	if strings.Contains(r.RequestURI, "/containers") {
		return states.ContainerOthers
	}
	if strings.Contains(r.RequestURI, "/images") && strings.HasSuffix(r.RequestURI, "/json") {
		return states.PassAsIs
	}
	if strings.HasSuffix(r.RequestURI, "/version") || strings.Contains(r.RequestURI, "/exec/"){
		return states.PassAsIs
	}
	if strings.Contains(r.RequestURI, "/volumes/create")  {
		return states.VolumeCreate
	}
	if strings.Contains(r.RequestURI, "/volumes")  {
		if r.Method == "DELETE" {
			return states.VolumeRemove
		}
		if r.Method == "GET" {
			if strings.HasSuffix(r.RequestURI,"/volumes") ||
			   strings.HasSuffix(r.RequestURI,"/volumes/") {
			   return states.VolumesList
			} else {
			   return states.VolumeInspect
			}
		}
	}
	return states.PassAsIs
}

//Init - Initialize the Validation and Handling APIs
func (*MultiTenant) Init() {
	log.Debug("In MultiTenant.Init()")
	if (os.Getenv("SWARM_MULTI_TENANT") == "false") {
		log.Debug("SWARM_MULTI_TENANT is false")
		return
	}
    if os.Getenv("SWARM_AUTH_BACKEND") == "Keystone" {
		log.Debug("SWARM_AUTH_BACKEND == Keystone")
		log.Debug("Keystone not supported")
//	    	aclsAPI = new(keystone.KeyStoneAPI)  	
	} else {
		if plugins.Len()<1{	
	    	log.Debug("SWARM_AUTH_BACKEND != Keystone")	
        	plugins.PushBack(handler.Handler(new(authN.DefaultAuthNImpl).Handle))
        	plugins.PushBack(handler.Handler(new(authZ.DefaultImp).Handle))
		}
	}
	log.Info("Init provision engine OK")
}
