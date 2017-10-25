package terminal

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/mattn/go-isatty"
)

var (
	singleArgFunctions = map[rune]func(int){
		'A': CursorUp,
		'B': CursorDown,
		'C': CursorForward,
		'D': CursorBack,
		'E': CursorNextLine,
		'F': CursorPreviousLine,
		'G': CursorHorizontalAbsolute,
	}
)

const (
	foregroundBlue      = 0x1
	foregroundGreen     = 0x2
	foregroundRed       = 0x4
	foregroundIntensity = 0x8
	foregroundMask      = (foregroundRed | foregroundBlue | foregroundGreen | foregroundIntensity)
	backgroundBlue      = 0x10
	backgroundGreen     = 0x20
	backgroundRed       = 0x40
	backgroundIntensity = 0x80
	backgroundMask      = (backgroundRed | backgroundBlue | backgroundGreen | backgroundIntensity)
)

type Writer struct {
	out     io.Writer
	handle  syscall.Handle
	orgAttr word
}

func NewAnsiStdout() io.Writer {
	var csbi consoleScreenBufferInfo
	out := os.Stdout
	if !isatty.IsTerminal(out.Fd()) {
		return out
	}
	handle := syscall.Handle(out.Fd())
	procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))
	return &Writer{out: out, handle: handle, orgAttr: csbi.attributes}
}

func NewAnsiStderr() io.Writer {
	var csbi consoleScreenBufferInfo
	out := os.Stderr
	if !isatty.IsTerminal(out.Fd()) {
		return out
	}
	handle := syscall.Handle(out.Fd())
	procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))
	return &Writer{out: out, handle: handle, orgAttr: csbi.attributes}
}

func (w *Writer) Write(data []byte) (n int, err error) {
	r := bytes.NewReader(data)

	for {
		ch, size, err := r.ReadRune()
		if err != nil {
			break
		}
		n += size

		switch ch {
		case '\x1b':
			size, err = w.handleEscape(r)
			n += size
			if err != nil {
				break
			}
		default:
			fmt.Fprint(w.out, string(ch))
		}
	}
	return
}

func (w *Writer) handleEscape(r *bytes.Reader) (n int, err error) {
	buf := make([]byte, 0, 10)
	buf = append(buf, "\x1b"...)

	// Check '[' continues after \x1b
	ch, size, err := r.ReadRune()
	if err != nil {
		fmt.Fprint(w.out, string(buf))
		return
	}
	n += size
	if ch != '[' {
		fmt.Fprint(w.out, string(buf))
		return
	}

	// Parse escape code
	var code rune
	argBuf := make([]byte, 0, 10)
	for {
		ch, size, err = r.ReadRune()
		if err != nil {
			fmt.Fprint(w.out, string(buf))
			return
		}
		n += size
		if ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') {
			code = ch
			break
		}
		argBuf = append(argBuf, string(ch)...)
	}

	w.applyEscapeCode(buf, string(argBuf), code)
	return
}

func (w *Writer) applyEscapeCode(buf []byte, arg string, code rune) {
	switch arg + string(code) {
	case "?25h":
		CursorShow()
		return
	case "?25l":
		CursorHide()
		return
	}

	if f, ok := singleArgFunctions[code]; ok {
		if n, err := strconv.Atoi(arg); err == nil {
			f(n)
			return
		}
	}

	switch code {
	case 'm':
		w.applySelectGraphicRendition(arg)
	default:
		buf = append(buf, string(code)...)
		fmt.Fprint(w.out, string(buf))
	}
}

// Original implementation: https://github.com/mattn/go-colorable
func (w *Writer) applySelectGraphicRendition(arg string) {
	if arg == "" {
		procSetConsoleTextAttribute.Call(uintptr(w.handle), uintptr(w.orgAttr))
		return
	}

	var csbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(uintptr(w.handle), uintptr(unsafe.Pointer(&csbi)))
	attr := csbi.attributes

	for _, param := range strings.Split(arg, ";") {
		n, err := strconv.Atoi(param)
		if err != nil {
			continue
		}

		switch {
		case n == 0 || n == 100:
			attr = w.orgAttr
		case 1 <= n && n <= 5:
			attr |= foregroundIntensity
		case 30 <= n && n <= 37:
			attr = (attr & backgroundMask)
			if (n-30)&1 != 0 {
				attr |= foregroundRed
			}
			if (n-30)&2 != 0 {
				attr |= foregroundGreen
			}
			if (n-30)&4 != 0 {
				attr |= foregroundBlue
			}
		case 40 <= n && n <= 47:
			attr = (attr & foregroundMask)
			if (n-40)&1 != 0 {
				attr |= backgroundRed
			}
			if (n-40)&2 != 0 {
				attr |= backgroundGreen
			}
			if (n-40)&4 != 0 {
				attr |= backgroundBlue
			}
		case 90 <= n && n <= 97:
			attr = (attr & backgroundMask)
			attr |= foregroundIntensity
			if (n-90)&1 != 0 {
				attr |= foregroundRed
			}
			if (n-90)&2 != 0 {
				attr |= foregroundGreen
			}
			if (n-90)&4 != 0 {
				attr |= foregroundBlue
			}
		case 100 <= n && n <= 107:
			attr = (attr & foregroundMask)
			attr |= backgroundIntensity
			if (n-100)&1 != 0 {
				attr |= backgroundRed
			}
			if (n-100)&2 != 0 {
				attr |= backgroundGreen
			}
			if (n-100)&4 != 0 {
				attr |= backgroundBlue
			}
		}
	}

	procSetConsoleTextAttribute.Call(uintptr(w.handle), uintptr(attr))
}
