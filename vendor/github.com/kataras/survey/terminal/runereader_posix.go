// +build !windows

// The terminal mode manipulation code is derived heavily from:
// https://github.com/golang/crypto/blob/master/ssh/terminal/util.go:
// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"bufio"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

type runeReaderState struct {
	term syscall.Termios
	buf  *bufio.Reader
}

func newRuneReaderState(input *os.File) runeReaderState {
	return runeReaderState{
		buf: bufio.NewReader(input),
	}
}

// For reading runes we just want to disable echo.
func (rr *RuneReader) SetTermMode() error {
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(rr.Input.Fd()), ioctlReadTermios, uintptr(unsafe.Pointer(&rr.state.term)), 0, 0, 0); err != 0 {
		return err
	}

	newState := rr.state.term
	newState.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG

	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(rr.Input.Fd()), ioctlWriteTermios, uintptr(unsafe.Pointer(&newState)), 0, 0, 0); err != 0 {
		return err
	}

	return nil
}

func (rr *RuneReader) RestoreTermMode() error {
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(rr.Input.Fd()), ioctlWriteTermios, uintptr(unsafe.Pointer(&rr.state.term)), 0, 0, 0); err != 0 {
		return err
	}
	return nil
}

func (rr *RuneReader) ReadRune() (rune, int, error) {
	r, size, err := rr.state.buf.ReadRune()
	if err != nil {
		return r, size, err
	}
	// parse ^[ sequences to look for arrow keys
	if r == '\033' {
		r, size, err = rr.state.buf.ReadRune()
		if err != nil {
			return r, size, err
		}
		if r != '[' {
			return r, size, fmt.Errorf("Unexpected Escape Sequence: %q", []rune{'\033', r})
		}
		r, size, err = rr.state.buf.ReadRune()
		if err != nil {
			return r, size, err
		}
		switch r {
		case 'D':
			return KeyArrowLeft, 1, nil
		case 'C':
			return KeyArrowRight, 1, nil
		case 'A':
			return KeyArrowUp, 1, nil
		case 'B':
			return KeyArrowDown, 1, nil
		}
		return r, size, fmt.Errorf("Unknown Escape Sequence: %q", []rune{'\033', '[', r})
	}
	return r, size, err
}
