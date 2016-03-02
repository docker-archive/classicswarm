package authZ

import (
	"bytes"
	"net/http"
	"strings"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/authZ/keystone"
	"github.com/docker/swarm/pkg/authZ/states"
	"io/ioutil"
	"github.com/docker/swarm/pkg/authZ/utils"
	"github.com/samalba/dockerclient"
	"encoding/json"
)

//Hooks - Entry point to AuthZ mechanisem
type Hooks struct{}

//TODO  - Hooks Infra for overriding swarm
//TODO  - Take bussiness logic out
//TODO  - Refactor packages
//TODO  - Expand API
//TODO -  Images...
//TODO - https://github.com/docker/docker/pull/15953
//TODO - https://github.com/docker/docker/pull/16331

var authZAPI HandleAuthZAPI
var aclsAPI ACLsAPI
//EventEnum - State of event
//type EventEnum int


//ApprovalEnum - State of approval
//type ApprovalEnum int

//PrePostAuthWrapper - Hook point from primary to the authZ mechanisem
func (*Hooks) PrePostAuthWrapper(cluster cluster.Cluster, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (os.Getenv("SWARM_MULTI_TENANT") == "false") {
			next.ServeHTTP(w, r)
			return
		}
		eventType := eventParse(r)
		defer r.Body.Close()
		
		//Bytes Json will be decoded into this one
        var containerConfig dockerclient.ContainerConfig
        var volumeCreateRequest dockerclient.VolumeCreateRequest
		reqBody, _ := ioutil.ReadAll(r.Body)
		if len(reqBody)== 0 {
			log.Debug("reqBody 0")
		} else {
			log.Debug("reqBody not 0")
			if eventType == states.ContainerCreate {
			  log.Debug("ContainerCreate")				
			  if err := json.NewDecoder(bytes.NewReader(reqBody)).Decode(&containerConfig); err != nil {
                log.Error(err)
                return
        		  }
			  log.Debugf("Requests containerConfig: %+v",containerConfig)
			} else if eventType == states.VolumeCreate {
			  log.Debug("VolumeCreate")
			  if err := json.NewDecoder(bytes.NewReader(reqBody)).Decode(&volumeCreateRequest); err != nil {
                log.Error(err)
                return
        		  }
			  log.Debugf("Requests volumeCreateRequest: %+v",volumeCreateRequest)
			}
		}


		r, e1 := utils.ModifyRequest(r, bytes.NewReader(reqBody), "", "")
		if e1 != nil {
			log.Error(e1)
		}
		
		log.Debug("*****modified*****r***************")
		log.Debug(r)
		log.Debug("*************************")


		isAllowed, dto := aclsAPI.ValidateRequest(cluster, eventType, w, r, reqBody, containerConfig)
//		isAllowed, dto := aclsAPI.ValidateRequest(cluster, eventType, r, containerConfig)
		if isAllowed == states.Admin {
			next.ServeHTTP(w, r)
			return
		}
		//TODO - all kinds of conditionals
		if eventType == states.PassAsIs || isAllowed == states.Approved || isAllowed == states.ConditionFilter {
			authZAPI.HandleEvent(eventType, w, r, next, dto, reqBody, containerConfig, volumeCreateRequest)
		} else {
			log.Debug("Return failure")
			http.Error(w, dto.ErrorMessage, http.StatusBadRequest)
			log.Debug("Returned failure")
		}
	})
}

/*Probably should use regular expressions here*/
func eventParse(r *http.Request) states.EventEnum {
	log.Debug("hooks.eventParse uri: ", r.RequestURI)

	if strings.Contains(r.RequestURI, "/containers") && (strings.Contains(r.RequestURI, "create")) {
		return states.ContainerCreate
	}

	if strings.Contains(r.RequestURI, "/containers/json") {
		return states.ContainersList
	}

	if strings.Contains(r.RequestURI, "/containers") &&
		(strings.Contains(r.RequestURI, "logs") || strings.Contains(r.RequestURI, "attach") || strings.HasSuffix(r.RequestURI, "exec")) {
		return states.StreamOrHijack
	}
	if strings.Contains(r.RequestURI, "/containers") && strings.HasSuffix(r.RequestURI, "/json") {
		return states.ContainerInspect
	}
	if strings.Contains(r.RequestURI, "/containers") {
		return states.ContainerOthers
	}
	if strings.Contains(r.RequestURI, "/images") && strings.HasSuffix(r.RequestURI, "/json") {
		return states.PassAsIs
	}
	if strings.HasSuffix(r.RequestURI, "/version") || strings.Contains(r.RequestURI, "/exec/"){
		return states.PassAsIs
	}



//	if strings.Contains(r.RequestURI, "Will add to here all APIs we explicitly want to block") {
//		return states.NotSupported
//	}

	return states.NotSupported
}

//Init - Initialize the Validation and Handling APIs
func (*Hooks) Init() {
	//TODO - should use a map for all the Pre . Post function like in primary.go
	log.Debug("In Hooks.Init()")
	if (os.Getenv("SWARM_MULTI_TENANT") == "false") {
		log.Debug("SWARM_MULTI_TENANT is false")
		return
	}
    if os.Getenv("SWARM_AUTH_BACKEND") == "Keystone" {
		log.Debug("SWARM_AUTH_BACKEND == Keystone")
	    	aclsAPI = new(keystone.KeyStoneAPI)  	
	} else {	
	    log.Debug("SWARM_AUTH_BACKEND != Keystone")	
		aclsAPI = new(DefaultACLsImpl)
	}
	aclsAPI.Init()	
	authZAPI = new(DefaultImp)
	authZAPI.Init()
	//TODO reflection using configuration file tring for the backend type

	log.Info("Init provision engine OK")
}
