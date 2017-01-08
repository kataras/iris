// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

// Package monotime provides functions to access monotonic clock source.
package monotime

import (
	"time"
	_ "unsafe" // required to use //go:linkname
)

//go:noescape
//go:linkname nanotime runtime.nanotime
func nanotime() int64

// Now returns the current time in nanoseconds from a monotonic clock.
//
// The time returned is based on some arbitrary platform-specific point in the
// past. The time returned is guaranteed to increase monotonically without
// notable jumps, unlike time.Now() from the Go standard library, which may
// jump forward or backward significantly due to system time changes or leap
// seconds.
//
// It's implemented using runtime.nanotime(), which uses CLOCK_MONOTONIC on
// Linux. Note that unlike CLOCK_MONOTONIC_RAW, CLOCK_MONOTONIC is affected
// by time changes. However, time changes never cause clock jumps; instead,
// clock frequency is adjusted slowly.
func Now() time.Duration {
	return time.Duration(nanotime())
}

// Since returns the time elapsed since t, obtained previously using Now.
func Since(t time.Duration) time.Duration {
	return Now() - t
}
