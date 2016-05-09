package authorization

import (
	"net/http"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authorization/states"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authorization/utils"
	"github.com/samalba/dockerclient"
)

//HandleAuthZAPI - API for backend ACLs services - for now only tenant seperation - finer grained later
type HandleAuthZAPI interface {

	//The Admin should first provision itself before starting to servce
	Init() error

//	HandleEvent(eventType states.EventEnum, w http.ResponseWriter, r *http.Request, next http.Handler, containerID *utils.ValidationOutPutDTO, reqBody []byte, containerConfig dockerclient.ContainerConfig)
	HandleEvent(eventType states.EventEnum, w http.ResponseWriter, r *http.Request, next http.Handler, containerID *utils.ValidationOutPutDTO, reqBody []byte, containerConfig dockerclient.ContainerConfig, volumeCreateRequest dockerclient.VolumeCreateRequest)
	Handle(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error
}
