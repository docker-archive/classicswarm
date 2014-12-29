// +build windows

package api

import (
	"crypto/tls"
	"fmt"
	"net"
)

func newUnixListener(addr string, tlsConfig *tls.Config) (net.Listener, error) {
	return nil, fmt.Errorf("Windows platform does not support a unix socket")
}
