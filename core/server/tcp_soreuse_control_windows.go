package server

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// Control sets the SO_REUSEADDR option on the socket.
func Control(network, address string, c syscall.RawConn) (err error) {
	return c.Control(func(fd uintptr) {
		err = windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_REUSEADDR, 1)
	})
}
