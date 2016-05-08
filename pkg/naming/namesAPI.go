package naming

import (
	"net/http"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/authZ/states"
)

type NamesAPI interface {
	Scope(cluster cluster.Cluster, eventType states.EventEnum, w http.ResponseWriter, r *http.Request, next http.Handler) error
}
