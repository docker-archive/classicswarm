package handler

import (
	"net/http"
	"github.com/docker/swarm/cluster"
)

type Handler func(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error