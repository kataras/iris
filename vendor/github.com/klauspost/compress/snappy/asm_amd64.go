//+build !noasm
//+build !appengine

// Copyright 2015, Klaus Post, see LICENSE for details.

package snappy

import (
	"github.com/klauspost/cpuid"
)

// matchLenSSE4 returns the number of matching bytes in a and b
// up to length 'max'. Both slices must be at least 'max'
// bytes in size.
// It uses the PCMPESTRI SSE 4.2 instruction.
//go:noescape
func matchLenSSE4(a, b []byte, max int) int

// Detect SSE 4.2 feature.
func init() {
	useSSE42 = cpuid.CPU.SSE42()
}

// matchLenSSE4Ref is a reference matcher.
func matchLenSSE4Ref(a, b []byte, max int) int {
	for i := 0; i < max; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return max
}
