//+build !amd64 noasm appengine

// Copyright 2015, Klaus Post, see LICENSE for details.

package snappy

func init() {
	useSSE42 = false
}

// matchLenSSE4 should never be called.
func matchLenSSE4(a, b []byte, max int) int {
	panic("no assembler")
	return 0
}
