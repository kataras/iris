package logger

import (
	"io"
)

// Logger is a simple interface which is used to log critical messages (aka panics).
type Logger func(errorMessage string) // don't touch it, if we add functions on Logger type,
// end-user will need to import the logger package to modify the logger and it's bad idea for a logger like this,
// do external functions instead like NewFrom....

// NewFromWriter returns a new Logger
// which writes to the writer.
func NewFromWriter(w io.Writer) Logger {
	return func(errorMessage string) {
		w.Write([]byte(errorMessage))
	}
}
