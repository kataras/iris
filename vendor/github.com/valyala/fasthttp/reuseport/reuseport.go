// +build linux darwin dragonfly freebsd netbsd openbsd

// Package reuseport provides TCP net.Listener with SO_REUSEPORT support.
//
// SO_REUSEPORT allows linear scaling server performance on multi-CPU servers.
// See https://www.nginx.com/blog/socket-sharding-nginx-release-1-9-1/ for more details :)
//
// The package is based on https://github.com/kavu/go_reuseport .
package reuseport

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
)

func getSockaddr(network, addr string) (sa syscall.Sockaddr, soType int, err error) {
	// TODO: add support for tcp and tcp6 networks.

	if network != "tcp4" {
		return nil, -1, errors.New("only tcp4 network is supported")
	}

	tcpAddr, err := net.ResolveTCPAddr(network, addr)
	if err != nil {
		return nil, -1, err
	}

	var sa4 syscall.SockaddrInet4
	sa4.Port = tcpAddr.Port
	copy(sa4.Addr[:], tcpAddr.IP.To4())
	return &sa4, syscall.AF_INET, nil
}

// ErrNoReusePort is returned if the OS doesn't support SO_REUSEPORT.
type ErrNoReusePort struct {
	err error
}

// Error implements error interface.
func (e *ErrNoReusePort) Error() string {
	return fmt.Sprintf("The OS doesn't support SO_REUSEPORT: %s", e.err)
}

// Listen returns TCP listener with SO_REUSEPORT option set.
//
// Only tcp4 network is supported.
//
// ErrNoReusePort error is returned if the system doesn't support SO_REUSEPORT.
func Listen(network, addr string) (l net.Listener, err error) {
	var (
		soType, fd int
		file       *os.File
		sockaddr   syscall.Sockaddr
	)

	if sockaddr, soType, err = getSockaddr(network, addr); err != nil {
		return nil, err
	}

	if fd, err = syscall.Socket(soType, syscall.SOCK_STREAM, syscall.IPPROTO_TCP); err != nil {
		return nil, err
	}

	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, soReusePort, 1); err != nil {
		syscall.Close(fd)
		return nil, &ErrNoReusePort{err}
	}

	if err = syscall.Bind(fd, sockaddr); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	if err = syscall.Listen(fd, syscall.SOMAXCONN); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	name := fmt.Sprintf("reuseport.%d.%s.%s", os.Getpid(), network, addr)
	file = os.NewFile(uintptr(fd), name)
	if l, err = net.FileListener(file); err != nil {
		file.Close()
		return nil, err
	}

	if err = file.Close(); err != nil {
		return nil, err
	}

	return l, err
}
