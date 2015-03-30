// +build !windows

package plugin

import (
	"net"
	"net/rpc"
	"os"
	"syscall"
)

func createEndpoint(name string) string {
	return "/tmp/" + STRATEGY_PLUGIN_PREFIX + name + ".sock"
}

func Listen(name string) (net.Listener, error) {
	addr := createEndpoint(name)
	if err := syscall.Unlink(addr); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	ln, err := net.Listen("unix", addr)
	return ln, err
}

func NewClient(name string) (*rpc.Client, error) {
	conn, err := net.Dial("unix", createEndpoint(name))
	if err != nil {
		return nil, err
	}
	client := rpc.NewClient(conn)
	return client, nil
}
