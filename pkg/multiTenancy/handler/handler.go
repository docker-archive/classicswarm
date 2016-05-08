package handler

import (
	"net/http"
	"container/list"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/authZ/states"
)

type Handler func(cluster cluster.Cluster, eventType states.EventEnum, w http.ResponseWriter, r *http.Request, next http.Handler, plugins *list.List) error