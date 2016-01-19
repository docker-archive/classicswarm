// +build !linux,!darwin

package cluster

import (
	"errors"
	"net"
	"time"
)

// setTCPUserTimeout doesn't work under Windows because Go doesn't support
// the option and swarm doesn't support cgo
// This is a usability enhancement. Service shouldn't fail on this error.
func setTCPUserTimeout(conn *net.TCPConn, uto time.Duration) error {
	return errors.New("Go doesn't have native support for TCP_USER_TIMEOUT for this platform")
}
