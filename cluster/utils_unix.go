// +build linux

package cluster

import (
	"net"
	"os"
	"syscall"
	"time"
)

// setTCPUserTimeout sets TCP_USER_TIMEOUT according to RFC5842
func setTCPUserTimeout(conn *net.TCPConn, uto time.Duration) error {
	f, err := conn.File()
	if err != nil {
		return err
	}
	defer f.Close()

	msecs := int(uto.Nanoseconds() / 1e6)
	// TCP_USER_TIMEOUT is a relatively new feature to detect dead peer from sender side.
	// Linux supports it since kernel 2.6.37. It's among Golang experimental under
	// golang.org/x/sys/unix but it doesn't support all Linux platforms yet.
	// we explicitly define it here until it becomes official in golang.
	// TODO: replace it with proper package when TCP_USER_TIMEOUT is supported in golang.
	const tcpUserTimeout = 0x12
	return os.NewSyscallError("setsockopt", syscall.SetsockoptInt(int(f.Fd()), syscall.IPPROTO_TCP, tcpUserTimeout, msecs))
}
