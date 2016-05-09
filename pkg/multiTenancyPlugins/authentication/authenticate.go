package authentication

import (
	"net/http"
	"os"
	"errors"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authorization/headers"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/pluginAPI"
	log "github.com/Sirupsen/logrus"
)

//AuthenticationImpl - implementation of plugin API
type AuthenticationImpl struct{
	nextHandler pluginAPI.Handler
}

func NewAuthentication(handler pluginAPI.Handler) pluginAPI.PluginAPI{
	authN := &AuthenticationImpl{
		nextHandler:     handler,
	}
	return authN
}

//Handle authentication on request and call next plugin handler.
func (authentication *AuthenticationImpl) Handle(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error {
	tenantIdToValidate := r.Header.Get(headers.AuthZTenantIdHeaderName)
	log.Debug("In AuthenticationImpl.Handle ...")
	if tenantIdToValidate == "" {
		return  errors.New("Not Authorized!")
	}
	if tenantIdToValidate == os.Getenv("SWARM_ADMIN_TENANT_ID") {
		return nil
	}
	err := authentication.nextHandler("", cluster, w, r, swarmHandler)
	if err != nil {
        	log.Error(err)
		return err
	}
	return nil
}
