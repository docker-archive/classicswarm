package auth

import (
	"fmt"

	"github.com/rackspace/gophercloud"
)

func init() {
	authProviders["rax"] = RaxAuthenticator
}

func raxAuthenticate(username, key string) (access *gophercloud.Access, err error) {
	provider := "https://identity.api.rackspacecloud.com/v2.0/tokens"
	authOptions := gophercloud.AuthOptions{
		Username: username,
		ApiKey:   key,
	}

	// Attempt to authenticate
	return gophercloud.Authenticate(provider, authOptions)
}

func RaxAuthenticator(env map[string]string) (err error) {
	if username, found := env[AUTH_USERNAME]; !found {
		return fmt.Errorf("[rax-auth] Auth failed. No user specified.\n")
	} else if apiKey, found := env[AUTH_KEY]; !found {
		return fmt.Errorf("[rax-auth] Auth failed. No API key specified.\n")
	} else if access, err := raxAuthenticate(username, apiKey); err == nil {
		if access == nil || access.AuthToken() == "" {
			return fmt.Errorf("Failed to authenticate user: %s\n", env["auth-user"])
		} else {
			// Remove the API key from the env passed along
			delete(env, AUTH_KEY)

			// Set our token ID
			env["auth-token"] = access.AuthToken()
			return nil
		}
	}

	return fmt.Errorf("[rax-auth] Auth failed. Bad credentials.\n")
}
