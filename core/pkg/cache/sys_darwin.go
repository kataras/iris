//go:build darwin
// +build darwin

package cache

import (
	"os/exec"
	"strconv"
	"strings"
)

// getTotalMemory returns the total available memory on macOS systems in bytes.
func getTotalMemory() (uint64, error) {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, err
	}

	memStr := strings.TrimSpace(string(out))
	mem, err := strconv.ParseUint(memStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return mem, nil
}
