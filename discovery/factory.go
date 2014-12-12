package discovery

import (
	"errors"
)

var (
	ErrDiscoveryUnknown = errors.New("Failed to find discovery backend for URL")
	ErrUnsetDiscovery   = errors.New("Discovery backend has not been set")
)

var discoveryBackends []DiscoveryBackend
var discoveryBackend DiscoveryBackend
var discoveryUrl string

// FetchSlaves returns the slaves for the discovery service at the specified endpoint
func FetchSlaves(token string) ([]string, error) {
	if discoveryBackend == nil || len(discoveryUrl) == 0 {
		return nil, ErrUnsetDiscovery
	}

	return discoveryBackend.FetchSlaves(discoveryUrl, token)
}

func RegisterSlave(addr, token string) error {
	if discoveryBackend == nil || len(discoveryUrl) == 0 {
		return ErrUnsetDiscovery
	}

	return discoveryBackend.RegisterSlave(discoveryUrl, addr, token)
}

// CreateCluster returns a unique cluster token
func CreateCluster() (string, error) {
	if discoveryBackend == nil || len(discoveryUrl) == 0 {
		return "", ErrUnsetDiscovery
	}

	return discoveryBackend.CreateCluster(discoveryUrl)
}

func SetupDiscovery(url string) (err error) {
	for _, backend := range discoveryBackends {
		var ok bool

		// Ignore errors from discovery unless no discovery backend is found
		if ok, err = backend.Supports(url); ok {
			discoveryBackend = backend
			discoveryUrl = url
			return nil
		}
	}

	if err == nil {
		err = ErrDiscoveryUnknown
	}

	return err
}

func RegisterBackend(backend DiscoveryBackend) {
	discoveryBackends = append(discoveryBackends, backend)
}

func init() {
	RegisterBackend(&DockerHubDiscovery{})
}
