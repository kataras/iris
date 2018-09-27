package terminal

import (
	"fmt"
)

var (
	Stdout = NewAnsiStdout()
)

// Print prints given arguments with escape sequence conversion for windows.
func Print(a ...interface{}) (n int, err error) {
	return fmt.Fprint(Stdout, a...)
}

// Printf prints a given format with escape sequence conversion for windows.
func Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(Stdout, format, a...)
}

// Println prints given arguments with newline and escape sequence conversion
// for windows.
func Println(a ...interface{}) (n int, err error) {
	return fmt.Fprintln(Stdout, a...)
}
