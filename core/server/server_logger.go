package server

import (
	"bytes"
	"io"
	"log"
)

func newServerLogger(w io.Writer, ignoreLogs ...string) *log.Logger {
	if w == nil {
		return nil
	}

	serverLogger := newServerLogWriter(w, ignoreLogs)
	return log.New(serverLogger, serverLoggerPrefix, 0)
}

const serverLoggerPrefix = "[HTTP Server] "

type serverLogWriter struct { // see #1875
	w          io.Writer
	ignoreLogs [][]byte
}

var newLineBytes = []byte("\n")

func newServerLogWriter(w io.Writer, ignoreLogs []string) *serverLogWriter {
	prefixAsByteSlice := []byte(serverLoggerPrefix)

	// build the ignore lines.
	ignoreLogsAsByteSlice := make([][]byte, 0, len(ignoreLogs))
	for _, s := range ignoreLogs {
		ignoreLogsAsByteSlice = append(ignoreLogsAsByteSlice, append(prefixAsByteSlice, []byte(s)...)) // append([]byte(s), newLineBytes...)
	}

	return &serverLogWriter{
		w:          w,
		ignoreLogs: ignoreLogsAsByteSlice,
	}
}

func (l *serverLogWriter) Write(p []byte) (int, error) {
	for _, ignoredLogBytes := range l.ignoreLogs {
		if bytes.Equal(bytes.TrimSuffix(p, newLineBytes), ignoredLogBytes) {
			return 0, nil
		}
	}

	return l.w.Write(p)
}
