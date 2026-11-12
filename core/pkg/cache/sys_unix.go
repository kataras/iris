//go:build linux

package cache

import (
	"syscall"
)

// getTotalMemory returns the total available memory on Linux systems in bytes.
func getTotalMemory() (uint64, error) {
	var info syscall.Sysinfo_t
	err := syscall.Sysinfo(&info)
	if err != nil {
		return 0, err
	}
	// Both operands are widened explicitly. Sysinfo_t.Totalram is a uint64 on a
	// 64-bit kernel but a uint32 on 386 and arm, so multiplying it by a
	// uint64(Unit) is a compile error there and this package — and with it the
	// whole framework — did not build for those targets at all.
	return uint64(info.Totalram) * uint64(info.Unit), nil
}
