// +build windows

package plugin

import (
	"github.com/natefinch/npipe"
	"net/rpc"
)

func createEndpoint(name string) string {
	return `\\.\pipe\` + STRATEGY_PLUGIN_PREFIX + name
}

func Listen(name string) (*npipe.PipeListener, error) {
	ln, err := npipe.Listen(createEndpoint(name))
	return ln, err
}

func NewClient(name string) (*rpc.Client, error) {
	conn, err := npipe.Dial(createEndpoint(name))
	if err != nil {
		return nil, err
	}
	client := rpc.NewClient(conn)
	return client, nil
}
