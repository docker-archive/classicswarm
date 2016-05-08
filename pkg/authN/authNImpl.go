package authN

import (
	"net/http"
	"os"
	"errors"
	//"container/list"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/authZ/headers"
	"github.com/docker/swarm/pkg/multiTenancy/handler"
//	"github.com/docker/swarm/pkg/naming"
	log "github.com/Sirupsen/logrus"
)

//DefaultAuthNImpl - Default implementation of Authentication API
type DefaultAuthNImpl struct{
	Next handler.Handler
}


func (authN DefaultAuthNImpl) Handle(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error {
	tenantIdToValidate := r.Header.Get(headers.AuthZTenantIdHeaderName)
	log.Debug("In AuthNImpl.handle ...")
	if tenantIdToValidate == "" {
		return  errors.New("Not Authorized!")
	}
	if tenantIdToValidate == os.Getenv("SWARM_ADMIN_TENANT_ID") {
		return nil
	}

    err := authN.Next("", cluster, w, r, swarmHandler)
    if err != nil {
        	log.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
    }
	return nil
}