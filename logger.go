// Copyright (c) 2016, Gerasimos Maropoulos and Go Authors using for the 'log,io,os' packages
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package iris

import (
	"io"
	"log"
	"os"
)

// LoggerOutTerminal os.Stdout , it's the default io.Writer to the Iris' logger
var LoggerOutTerminal = os.Stdout

// Logger is just a log.Logger
type Logger struct {
	Logger  *log.Logger
	enabled bool
}

// NewLogger creates a new Logger.   The out variable sets the
// destination to which log data will be written.
// The prefix appears at the beginning of each generated log line.
// The flag argument defines the logging properties.
func NewLogger(out io.Writer, prefix string, flag int) *Logger {
	if out == nil {
		out = LoggerOutTerminal
	}
	return &Logger{Logger: log.New(out, LoggerIrisPrefix+prefix, flag), enabled: true}
}

// SetEnable true enables, false disables the Logger
func (l *Logger) SetEnable(enable bool) {
	l.enabled = enable
}

// IsEnabled returns true if Logger is enabled, otherwise false
func (l *Logger) IsEnabled() bool {
	return l.enabled
}

// Print calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Print(v ...interface{}) {
	if l.enabled {
		l.Logger.Print(v...)
	}
}

// Printf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Printf(format string, a ...interface{}) {
	if l.enabled {
		l.Logger.Printf(format, a...)
	}
}

// Println calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Println(a ...interface{}) {
	if l.enabled {
		l.Logger.Println(a...)
	}
}

// Fatal is equivalent to l.Print() followed by a call to os.Exit(1).
func (l *Logger) Fatal(a ...interface{}) {
	if l.enabled {
		l.Logger.Fatal(a...)
	} else {
		os.Exit(1) //we have to exit at any case because this is the Fatal
	}

}

// Fatalf is equivalent to l.Printf() followed by a call to os.Exit(1).
func (l *Logger) Fatalf(format string, a ...interface{}) {
	if l.enabled {
		l.Logger.Fatalf(format, a...)
	} else {
		os.Exit(1)
	}
}

// Fatalln is equivalent to l.Println() followed by a call to os.Exit(1).
func (l *Logger) Fatalln(a ...interface{}) {
	if l.enabled {
		l.Logger.Fatalln(a...)
	} else {
		os.Exit(1)
	}
}

// Panic is equivalent to l.Print() followed by a call to panic().
func (l *Logger) Panic(a ...interface{}) {
	if l.enabled {
		l.Logger.Panic(a...)
	}
}

// Panicf is equivalent to l.Printf() followed by a call to panic().
func (l *Logger) Panicf(format string, a ...interface{}) {
	if l.enabled {
		l.Logger.Panicf(format, a...)
	}
}

// Panicln is equivalent to l.Println() followed by a call to panic().
func (l *Logger) Panicln(a ...interface{}) {
	if l.enabled {
		l.Logger.Panicln(a...)
	}
}
