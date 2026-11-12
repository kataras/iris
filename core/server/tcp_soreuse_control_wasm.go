//go:build wasm
// +build wasm

package server

import "syscall"

// Control is a no-op on wasm.
func Control(network, address string, c syscall.RawConn) error {
	return nil
}
