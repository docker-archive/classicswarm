package authN

import (
	"net/http"
	"github.com/docker/swarm/cluster"
)

type AuthNAPI interface {
//	Authenticate(cluster cluster.Cluster, eventType states.EventEnum, w http.ResponseWriter, r *http.Request, next http.Handler) error
	Handle(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error
}
