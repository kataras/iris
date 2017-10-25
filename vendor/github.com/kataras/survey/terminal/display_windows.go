package terminal

import (
	"os"
	"syscall"
	"unsafe"
)

func EraseLine(mode EraseLineMode) {
	handle := syscall.Handle(os.Stdout.Fd())

	var csbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))

	var w uint32
	var x Short
	cursor := csbi.cursorPosition
	switch mode {
	case ERASE_LINE_END:
		x = csbi.size.X
	case ERASE_LINE_START:
		x = 0
	case ERASE_LINE_ALL:
		cursor.X = 0
		x = csbi.size.X
	}
	procFillConsoleOutputCharacter.Call(uintptr(handle), uintptr(' '), uintptr(x), uintptr(*(*int32)(unsafe.Pointer(&cursor))), uintptr(unsafe.Pointer(&w)))
}
