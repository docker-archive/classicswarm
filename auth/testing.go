package auth

import (
	"fmt"
)

func init() {
	authProviders["fail"] = FailingAuthenticator
	authProviders["pass"] = PassingAuthenticator
}

func FailingAuthenticator(args map[string]string) (err error) {
	if _, found := args["auth-user"]; !found {
		return fmt.Errorf("[fail-test] Auth failed. No user specified.\n")
	}

	return fmt.Errorf("[inline-auth] Auth failed. Bad credentials.\n")
}

func PassingAuthenticator(args map[string]string) (err error) {
	// Remote important things
	delete(args, AUTH_KEY)
	delete(args, AUTH_PASSWORD)
	return
}
