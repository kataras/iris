// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logger

import (
	"fmt"
	"io"
)

// writerFunc is just an "extended" io.Writer which provides
// some "methods" which can help applications
// to adapt their existing loggers inside Iris.
type writerFunc func(p []byte) (n int, err error)

func (w writerFunc) Write(p []byte) (int, error) {
	return w(p)
}

type formatPrintWriter interface {
	Printf(string, ...interface{})
}

type stringWriter interface {
	WriteString(string) (int, error)
}

// Log sends a message to the defined io.Writer logger, it's
// just a help function for internal use but it can be used to a cusotom middleware too.
//
// See AttachLogger too.
func Log(w io.Writer, format string, a ...interface{}) {
	// check if the user's defined logger is one of the "high"
	// level printers, if yes then use their functions to print instead
	// of allocating new byte slices.

	if fpw, ok := w.(formatPrintWriter); ok {
		fpw.Printf(format, a...)
		return
	}
	formattedMessage := fmt.Sprintf(format, a...)

	if sWriter, ok := w.(stringWriter); ok {
		sWriter.WriteString(formattedMessage)
		return
	}

	w.Write([]byte(formattedMessage))
}
