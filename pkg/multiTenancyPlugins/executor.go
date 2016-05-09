package multiTenancyPlugins

import (
	"net/http"
	"os"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authentication"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authorization"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/pluginAPI"
)

//Executor - Entry point to multi-tenancy plugins
type Executor struct{}

var startHandler pluginAPI.Handler

//Handle - Hook point from primary to plugins
func (*Executor) Handle(cluster cluster.Cluster, swarmHandler http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	log.Debug("In multiTenant.Handle ...")	
	if (os.Getenv("SWARM_MULTI_TENANT") == "false") {
	    swarmHandler.ServeHTTP(w, r)
	    return
	}
	err := startHandler("", cluster, w, r, swarmHandler)
        if err != nil {
            log.Error(err)
	    http.Error(w, err.Error(), http.StatusBadRequest)
	}
    })
}


//Init - Initialize the Validation and Handling plugins
func (*Executor) Init() {
    log.Debug("In MultiTenant.Init()")
    if (os.Getenv("SWARM_MULTI_TENANT") == "false") {
	log.Debug("SWARM_MULTI_TENANT is false")
	return
    }
    if os.Getenv("SWARM_AUTH_BACKEND") == "Keystone" {
	log.Debug("Keystone not supported")	
    } else {
	authorizationPlugin := new(authorization.DefaultImp)
	authenticationPlugin := authentication.NewAuthentication(authorizationPlugin.Handle)
	startHandler = authenticationPlugin.Handle
    }
}
