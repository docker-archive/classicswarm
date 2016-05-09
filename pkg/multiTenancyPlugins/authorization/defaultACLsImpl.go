package authorization

import (
	"os"
	"net/http"
	"github.com/docker/swarm/cluster"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authorization/states"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authorization/headers"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authorization/utils"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authorization/flavors"
	"github.com/samalba/dockerclient"
)

//DefaultACLsImpl - Default implementation of ACLs API
type DefaultACLsImpl struct{}

/*
ValidateRequest - Who wants to do what - allow or not
*/
func (*DefaultACLsImpl) ValidateRequest(cluster cluster.Cluster, eventType states.EventEnum, w http.ResponseWriter, r *http.Request, reqBody []byte, containerConfig dockerclient.ContainerConfig) (states.ApprovalEnum, *utils.ValidationOutPutDTO) {
	tenantIdToValidate := r.Header.Get(headers.AuthZTenantIdHeaderName)
	log.Debug("**DefaultACLsImpl.ValidateRequest***")

	if tenantIdToValidate == "" {
		return states.NotApproved, &utils.ValidationOutPutDTO{ErrorMessage: "Not Authorized!"}
	}
	if tenantIdToValidate == os.Getenv("SWARM_ADMIN_TENANT_ID") {
		return states.Admin, nil
	}
	//TODO - Duplication revise
	switch eventType {
	case states.ContainerCreate:
	    if (!flavors.IsFlavorValid(containerConfig)) {
			return states.NotApproved,&utils.ValidationOutPutDTO{ErrorMessage: "No flavor matches resource request!"}
		}
		valid, dto := utils.CheckContainerReferences(cluster, tenantIdToValidate, containerConfig)
		var err error
        if valid {
		  if dto.Binds,err = utils.CheckVolumeBinds(tenantIdToValidate, containerConfig); err != nil {
			valid = false
			dto.ErrorMessage = err.Error()
		  }
		}
		log.Debug(valid)
		log.Debugf("dto %+v",dto)
		if !valid {
			return states.NotApproved, dto			
		} 
		return states.Approved, dto
	case states.ContainersList:
		return states.ConditionFilter, nil
	case states.Unauthorized:
		return states.Approved, &utils.ValidationOutPutDTO{ErrorMessage: "Not Authorized!"}
	case states.VolumeCreate:
		return states.Approved, nil
	case states.VolumesList:
		return states.Approved, nil
	case states.VolumeInspect:
		return states.Approved, nil
	case states.VolumeRemove:
		return states.Approved, nil
	default:
		//CONTAINER_INSPECT / CONTAINER_OTHERS / STREAM_OR_HIJACK / PASS_AS_IS
		isOwner, dto := utils.CheckOwnerShip(cluster, tenantIdToValidate, r)
		if isOwner {
			return states.Approved, dto
		}
	}
	return states.NotApproved, &utils.ValidationOutPutDTO{ErrorMessage: "[Not authorized or no such resource!]"}
}

//Init - Any required initialization
func (*DefaultACLsImpl) Init() error {
	return nil
}
	
