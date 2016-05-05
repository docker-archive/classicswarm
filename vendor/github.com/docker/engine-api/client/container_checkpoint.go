package client

import (
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
)

// ContainerCheckpoint checkpoints a running container
func (cli *Client) ContainerCheckpoint(ctx context.Context, containerID string, options types.CriuConfig) error {
	resp, err := cli.post(ctx, "/containers/"+containerID+"/checkpoint", nil, options, nil)
	ensureReaderClosed(resp)

	return err
}
