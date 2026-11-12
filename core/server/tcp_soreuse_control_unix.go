//go:build !windows && !wasm
// +build !windows,!wasm

package server

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// Control sets the SO_REUSEADDR and SO_REUSEPORT options on the socket.
func Control(network, address string, c syscall.RawConn) (err error) {
	c.Control(func(fd uintptr) {
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		if err != nil {
			return
		}

		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		if err != nil {
			return
		}
	})

	return
}
