package multiTenancyPlugins

import (
	"net/http"
	"os"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/quota"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authentication"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authorization"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/apifilter"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/flavors"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/naming"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/keystone"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/pluginAPI"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/utils"
)

//Executor - Entry point to multi-tenancy plugins
type Executor struct{}

var startHandler pluginAPI.Handler

//Handle - Hook point from primary to plugins
func (*Executor) Handle(cluster cluster.Cluster, swarmHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debug(r)
		err := startHandler(utils.ParseCommand(r), cluster, w, r, swarmHandler)
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})
}

//Init - Initialize the Validation and Handling plugins
func (*Executor) Init() {
	if os.Getenv("SWARM_MULTI_TENANT") == "false" {
		log.Debug("SWARM_MULTI_TENANT is false")
		return
	}
	quotaPlugin := quota.NewQuota(nil)
	authorizationPlugin := authorization.NewAuthorization(quotaPlugin.Handle)
	nameScoping := namescoping.NewNameScoping(authorizationPlugin.Handle)
	flavorsPlugin := flavors.NewPlugin(nameScoping.Handle)
	apiFilterPlugin := apifilter.NewPlugin(flavorsPlugin.Handle)
	authenticationPlugin := authentication.NewAuthentication(apiFilterPlugin.Handle)
	keystonePlugin := keystone.NewPlugin(authenticationPlugin.Handle)
	startHandler = keystonePlugin.Handle
}
