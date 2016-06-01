package authentication

import (
	"errors"
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/headers"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/pluginAPI"
)

//AuthenticationImpl - implementation of plugin API
type AuthenticationImpl struct {
	nextHandler pluginAPI.Handler
}

func NewAuthentication(handler pluginAPI.Handler) pluginAPI.PluginAPI {
	authN := &AuthenticationImpl{
		nextHandler: handler,
	}
	return authN
}

//Handle authentication on request and call next plugin handler.
func (authentication *AuthenticationImpl) Handle(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error {
	log.Debug("Plugin authN got command: " + command)
	tenantIdToValidate := r.Header.Get(headers.AuthZTenantIdHeaderName)
	if tenantIdToValidate == "" {
		return errors.New("Not Authorized!")
	}
	if tenantIdToValidate == os.Getenv("SWARM_ADMIN_TENANT_ID") {
		swarmHandler.ServeHTTP(w, r)
		return nil
	}
	return authentication.nextHandler(command, cluster, w, r, swarmHandler)
}
