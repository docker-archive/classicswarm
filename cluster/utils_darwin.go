// +build darwin

package dockerclient

import (
	"net"
	"os"
	"syscall"
	"time"
)

// setTCPUserTimeout sets TCP_RXT_CONNDROPTIME on darwin
func setTCPUserTimeout(conn *net.TCPConn, uto time.Duration) error {
	f, err := conn.File()
	if err != nil {
		return err
	}
	defer f.Close()

	secs := int(uto.Nanoseconds() / 1e9)
	return os.NewSyscallError("setsockopt", syscall.SetsockoptInt(int(f.Fd()), syscall.IPPROTO_TCP, unix.TCP_RXT_CONNDROPTIME, secs))
}
