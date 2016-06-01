package client

import (
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
)

// ContainerMigrate migrates a container from the given container with the given name to another node
func (cli *Client) ContainerMigrate(ctx context.Context, containerID string, filters types.MigrateFiltersConfig) error {
	resp, err := cli.post(ctx, "/containers/"+containerID+"/migrate", nil, filters, nil)
	ensureReaderClosed(resp)
	return err
}
