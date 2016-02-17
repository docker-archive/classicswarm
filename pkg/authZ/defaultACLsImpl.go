package authZ

import (
	//	"bytes"
	//	"io/ioutil"
	"net/http"

	"github.com/docker/swarm/cluster"
	//	"github.com/docker/swarm/cluster/swarm"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/pkg/authZ/states"
	//	"github.com/docker/swarm/cluster/swarm"
	"github.com/docker/swarm/pkg/authZ/headers"
	"github.com/docker/swarm/pkg/authZ/utils"
	"github.com/docker/swarm/pkg/authZ/flavors"
	"github.com/samalba/dockerclient"
)

//DefaultACLsImpl - Default implementation of ACLs API
type DefaultACLsImpl struct{}

/*
ValidateRequest - Who wants to do what - allow or not
*/
func (*DefaultACLsImpl) ValidateRequest(cluster cluster.Cluster, eventType states.EventEnum, w http.ResponseWriter, r *http.Request, reqBody []byte, containerConfig dockerclient.ContainerConfig) (states.ApprovalEnum, *utils.ValidationOutPutDTO) {
//func (*DefaultACLsImpl) ValidateRequest(cluster cluster.Cluster, eventType states.EventEnum, r *http.Request, containerConfig dockerclient.ContainerConfig) (states.ApprovalEnum, *utils.ValidationOutPutDTO) {
	tenantIdToValidate := r.Header.Get(headers.AuthZTenantIdHeaderName)
	log.Debug("**ValidateRequest***")

	if tenantIdToValidate == "" {
		return states.NotApproved, &utils.ValidationOutPutDTO{ErrorMessage: "Not Authorized!"}
	}
	//TODO - Duplication revise
	switch eventType {
	case states.ContainerCreate:
	    if (!flavors.IsFlavorValid(containerConfig)) {
			return states.NotApproved,&utils.ValidationOutPutDTO{ErrorMessage: "No flavor matches resource request!"}
	
		}
		valid, dto := utils.CheckLinksOwnerShip(cluster, tenantIdToValidate, containerConfig)
		log.Debug(valid)
		log.Debugf("dto %+v",dto)
		log.Debug("-----------------")
		if !valid {
			return states.NotApproved, dto			
		} 
		return states.Approved, dto
	case states.ContainersList:
		return states.ConditionFilter, nil
	case states.Unauthorized:
		return states.NotApproved, &utils.ValidationOutPutDTO{ErrorMessage: "Not Authorized!"}
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
	
