package authN

import (
	"net/http"
	"container/list"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/authZ/states"
)

type AuthNAPI interface {
//	Authenticate(cluster cluster.Cluster, eventType states.EventEnum, w http.ResponseWriter, r *http.Request, next http.Handler) error
	Handle(cluster cluster.Cluster, eventType states.EventEnum, w http.ResponseWriter, r *http.Request, next http.Handler, plugins *list.List) error
}
