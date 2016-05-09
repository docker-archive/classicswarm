package pluginAPI

import (
	"net/http"
	"github.com/docker/swarm/cluster"
)

//This type define plugin entry function signature.
type Handler func(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error

type PluginAPI interface {
	Handle(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error
}