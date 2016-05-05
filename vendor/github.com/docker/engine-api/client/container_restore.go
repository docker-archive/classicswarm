package client

import (
	"net/url"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
)

// ContainerRestore restores a running container
func (cli *Client) ContainerRestore(ctx context.Context, containerID string, options types.CriuConfig, forceRestore bool) error {
	query := url.Values{}

	if forceRestore {
		query.Set("force", "1")
	}

	resp, err := cli.post(ctx, "/containers/"+containerID+"/restore", query, options, nil)
	ensureReaderClosed(resp)

	return err
}
