package authN

import (
	"net/http"
	"os"
	"errors"
	"github.com/docker/swarm/pkg/authZ/headers"
//	"github.com/docker/swarm/pkg/authZ/states"
//	"github.com/docker/swarm/pkg/authZ/utils"
	log "github.com/Sirupsen/logrus"
)

//DefaultAuthNImpl - Default implementation of Authentication API
type DefaultAuthNImpl struct{}

func (authN DefaultAuthNImpl) Authenticate(r *http.Request) error {
	tenantIdToValidate := r.Header.Get(headers.AuthZTenantIdHeaderName)
	log.Debug("**DefaultACLsImpl.ValidateRequest***")

	if tenantIdToValidate == "" {
		return  errors.New("Not Authorized!")
//		return states.NotApproved, &utils.ValidationOutPutDTO{ErrorMessage: "Not Authorized!"}
	}
	if tenantIdToValidate == os.Getenv("SWARM_ADMIN_TENANT_ID") {
		return nil
//		return states.Admin, nil
	}
	return nil
}
