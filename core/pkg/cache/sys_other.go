//go:build !linux && !darwin && !windows

package cache

import "errors"

// errTotalMemoryUnsupported is returned on platforms with no implementation here.
var errTotalMemoryUnsupported = errors.New("cache: total memory is unavailable on this platform")

// getTotalMemory reports that it cannot measure memory on this platform.
//
// Only linux, darwin and windows had an implementation, and without this file
// the package simply did not compile anywhere else — the BSDs, Solaris, AIX and
// js/wasm all failed on an undefined getTotalMemory rather than degrading.
//
// Returning an error rather than a guess is safe because it is the case both
// callers already handle: calculateDefaultMaxCount and calculateItemsToRemove
// fall back to a fixed budget when the lookup fails, which is exactly what a
// failing syscall gives them on a supported platform. Reaching for gopsutil
// instead would measure it properly, but it would also put a third-party
// dependency underneath core/, which the per-platform syscalls here exist to
// avoid.
func getTotalMemory() (uint64, error) {
	return 0, errTotalMemoryUnsupported
}
