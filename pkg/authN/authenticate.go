package authN

import "net/http"

type AuthNAPI interface {
	Authenticate(r *http.Request) error
}
