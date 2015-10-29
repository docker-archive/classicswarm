package authZ

import (
	"net/http"

	"github.com/docker/swarm/cluster"
)

//API for backend ACLs services - for now only tenant seperation - finer grained later
type ACLsAPI interface {

	//The Admin should first provision itself before starting to servce
	Init() error

	//Is valid and the label for the token if it is valid.
	//TODO - expand response according to design
	ValidateRequest(cluster cluster.Cluster, eventType EVENT_ENUM, w http.ResponseWriter, r *http.Request) (APPROVAL_ENUM, string)
}
