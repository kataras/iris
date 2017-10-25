// +build !windows

package terminal

import (
	"fmt"
)

func EraseLine(mode EraseLineMode) {
	fmt.Printf("\x1b[%dK", mode)
}
