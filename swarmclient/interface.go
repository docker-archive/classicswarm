package swarmclient

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

// SwarmAPIClient contains the subset of the docker/api interface relevant to Docker Swarm
type SwarmAPIClient interface {
	client.ContainerAPIClient
	client.ImageAPIClient
	client.NetworkAPIClient
	client.SystemAPIClient
	client.VolumeAPIClient
	ClientVersion() string
	NegotiateAPIVersion(ctx context.Context)
	ServerVersion(ctx context.Context) (types.Version, error)
}
