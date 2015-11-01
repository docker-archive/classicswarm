package authZ

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
)

//TODO  - Infra for overriding swarm
//TODO  - Take bussiness logic out
//TODO  - Refactor packages
//TODO  - Expand API
//TODO -  Images...
//TODO - https://github.com/docker/docker/pull/15953
//TODO - https://github.com/docker/docker/pull/16331
type Hooks struct{}

var authZAPI AuthZAPI
var aclsAPI ACLsAPI

type EVENT_ENUM int
type APPROVAL_ENUM int

func (*Hooks) PrePostAuthWrapper(cluster cluster.Cluster, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eventType := eventParse(r)
		allowed, containerId := aclsAPI.ValidateRequest(cluster, eventType, w, r)
		//TODO - all kinds of conditionals
		if eventType == PASS_AS_IS || allowed == APPROVED || allowed == CONDITION_FILTER {
			authZAPI.HandleEvent(eventType, w, r, next, containerId)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Not Authorized!"))
		}
	})
}

func eventParse(r *http.Request) EVENT_ENUM {
	log.Debug("Got the uri...", r.RequestURI)

	if strings.Contains(r.RequestURI, "/containers") && (strings.Contains(r.RequestURI, "create")) {
		return CONTAINER_CREATE
	}

	if strings.Contains(r.RequestURI, "/containers/json") {
		return CONTAINERS_LIST
	}

	if strings.Contains(r.RequestURI, "/containers") &&
		(strings.Contains(r.RequestURI, "logs") || strings.Contains(r.RequestURI, "attach") || strings.Contains(r.RequestURI, "exec")) {
		return STREAM_OR_HIJACK
	}
	if strings.Contains(r.RequestURI, "/containers") && strings.HasSuffix(r.RequestURI, "/json") {
		return CONTAINER_INSPECT
	}
	if strings.Contains(r.RequestURI, "/containers") {
		return CONTAINER_OTHERS
	}

	if strings.Contains(r.RequestURI, "Will add to here all APIs we explicitly want to block") {
		return NOT_SUPPORTED
	}

	return PASS_AS_IS
}

func (*Hooks) Init() {
	//TODO - should use a map for all the Pre . Post function like in primary.go

	aclsAPI = new(DefaultACLsImpl)
	authZAPI = new(DefaultImp)
	//TODO reflection using configuration file tring for the backend type

	log.Info("Init provision engine OK")
}
