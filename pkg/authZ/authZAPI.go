package authZ

import (
	"net/http"
)

//API for backend ACLs services - for now only tenant seperation - finer grained later
type AuthZAPI interface {

	//The Admin should first provision itself before starting to servce
	Init() error

	HandleEvent(eventType EVENT_ENUM, w http.ResponseWriter, r *http.Request, next http.Handler, containerId string)
}
