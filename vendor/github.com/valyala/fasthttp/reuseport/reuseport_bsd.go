// +build darwin dragonfly freebsd netbsd openbsd

package reuseport

import (
	"syscall"
)

const soReusePort = syscall.SO_REUSEPORT
