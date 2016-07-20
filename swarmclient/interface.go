package swarmclient

import (
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
)

// SwarmAPIClient contains the subset of the engine-api interface relevant to Docker Swarm
type SwarmAPIClient interface {
	client.ContainerAPIClient
	client.ImageAPIClient
	client.NetworkAPIClient
	client.SystemAPIClient
	client.VolumeAPIClient
	ClientVersion() string
	ServerVersion(ctx context.Context) (types.Version, error)
	UpdateClientVersion(v string)
}
