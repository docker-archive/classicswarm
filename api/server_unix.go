// +build !windows

package api

import (
	"crypto/tls"
	"net"
	"os"
	"syscall"
)

func newUnixListener(addr string, tlsConfig *tls.Config) (net.Listener, error) {
	if err := syscall.Unlink(addr); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// there is no way to specify the unix rights to use when
	// creating the socket with net.Listener, so we use umask
	// to create the file without rights and then we chmod
	// to the desired unix rights. This prevent unwanted
	// connections between the creation and the chmod
	mask := syscall.Umask(0777)
	defer syscall.Umask(mask)

	l, err := newListener("unix", addr, tlsConfig)
	if err != nil {
		return nil, err
	}

	// only usable by the user who started swarm
	if err := os.Chmod(addr, 0600); err != nil {
		return nil, err
	}

	return l, nil
}
