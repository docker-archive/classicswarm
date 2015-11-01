package authZ

import (
	//	"bytes"
	//	"io/ioutil"
	//	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	//	"github.com/docker/swarm/cluster/swarm"
	"strings"

	"github.com/gorilla/mux"
)

type DefaultACLsImpl struct{}


var authZTokenHeaderName string = "X-Auth-Token"
var tenancyLabel string = "com.swarm.tenant.0"

/*
Who wants to do what - allow or not
*/
func (*DefaultACLsImpl) ValidateRequest(cluster cluster.Cluster, eventType EVENT_ENUM, w http.ResponseWriter, r *http.Request) (APPROVAL_ENUM, string) {
	tokenToValidate := r.Header.Get(authZTokenHeaderName)

	if tokenToValidate == "" {
		return NOT_APPROVED, ""
	}
	//TODO - Duplication revise
	switch eventType {
	case CONTAINER_CREATE:
		return APPROVED, ""
	case CONTAINERS_LIST:
		return CONDITION_FILTER, ""
	case UNAUTHORIZED:
		return NOT_APPROVED, ""
	default:
		//CONTAINER_INSPECT / CONTAINER_OTHERS / STREAM_OR_HIJACK / PASS_AS_IS
		isOwner, id := checkOwnerShip(cluster, tokenToValidate, r)
		if isOwner {
			return APPROVED, id
		}
	}
	return NOT_APPROVED, ""
}

//TODO - Pass by ref ?
func checkOwnerShip(cluster cluster.Cluster, tenantName string, r *http.Request) (bool, string) {
	containers := cluster.Containers()
	log.Debug("got name: ", mux.Vars(r)["name"])
	tenantSet := make(map[string]bool)
	for _, container := range containers {
		if "/"+mux.Vars(r)["name"]+tenantName == container.Info.Name {
			log.Debug("Match By name!")
			return true, container.Info.Id
		} else if mux.Vars(r)["name"] == container.Info.Id {
			log.Debug("Match By full ID! Checking Ownership...")
			log.Debug("Tenant name: ", tenantName)
			log.Debug("Tenant Lable: ", container.Labels[tenancyLabel])
			if container.Labels[tenancyLabel] == tenantName {
				return true, container.Info.Id
			} else {
				return false, ""
			}
		}
		if container.Labels[tenancyLabel] == tenantName {
			tenantSet[container.Id] = true
		}
	}

	//Handle short ID
	ambiguityCounter := 0
	var returnID string
	for k := range tenantSet {
		if strings.HasPrefix(cluster.Container(k).Info.Id, mux.Vars(r)["name"]) {
			ambiguityCounter++
			returnID = cluster.Container(k).Info.Id
		}
		if ambiguityCounter == 1 {
			log.Debug("Matched by short ID")
			return true, returnID
		}
		if ambiguityCounter > 1 {
			log.Debug("Ambiguiy by short ID")
			//TODO - ambiguity
		}
		if ambiguityCounter == 0 {
			log.Debug("No match by short ID")
			//TODO - no such container
		}
	}
	return false, ""
}

func (*DefaultACLsImpl) Init() error {
	return nil
}
