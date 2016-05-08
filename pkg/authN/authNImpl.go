package authN

import (
	"net/http"
	"os"
	"errors"
	"container/list"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/authZ/headers"
	"github.com/docker/swarm/pkg/multiTenancy/handler"
	"github.com/docker/swarm/pkg/authZ/states"
//	"github.com/docker/swarm/pkg/naming"
	log "github.com/Sirupsen/logrus"
)

//DefaultAuthNImpl - Default implementation of Authentication API
type DefaultAuthNImpl struct{}

//func (authN DefaultAuthNImpl) Authenticate(cluster cluster.Cluster, eventType states.EventEnum, w http.ResponseWriter, r *http.Request, next http.Handler) error {
//	tenantIdToValidate := r.Header.Get(headers.AuthZTenantIdHeaderName)
//	log.Debug("In AuthNImpl.Authenticate ...")
//	if tenantIdToValidate == "" {
//		return  errors.New("Not Authorized!")
//	}
//	if tenantIdToValidate == os.Getenv("SWARM_ADMIN_TENANT_ID") {
//		return nil
//	}
//	//nameScope := naming.DefaultNamesImpl
//	//err := nameScope.Scope(cluster, eventType, w, r, next)// call next(naming) plugin	
//	return nil
//}

func (authN DefaultAuthNImpl) Handle(cluster cluster.Cluster, eventType states.EventEnum, w http.ResponseWriter, r *http.Request, next http.Handler, plugins *list.List) error {
	tenantIdToValidate := r.Header.Get(headers.AuthZTenantIdHeaderName)
	log.Debug("In AuthNImpl.handle ...")
	log.Debug(plugins.Len())
	if tenantIdToValidate == "" {
		return  errors.New("Not Authorized!")
	}
	if tenantIdToValidate == os.Getenv("SWARM_ADMIN_TENANT_ID") {
		return nil
	}

	plugin := plugins.Front()
    plugins.Remove(plugins.Front())
    err := plugin.Value.(handler.Handler)(cluster, eventType, w, r, next, plugins)
    if err != nil {
        	log.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
    }

	return nil
}