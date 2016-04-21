// Copyright (c) 2016, Gerasimos Maropoulos
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
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
///TODO: write all errors from iris package here and do some linting to all files

package iris

import (
	"fmt"
)

var (
	// Server

	// ErrServerPortAlreadyUsed returns an error with message: 'Server can't run, port is already used'
	ErrServerPortAlreadyUsed = NewError("Server can't run, port is already used")
	// ErrServerAlreadyStarted returns an error with message: 'Server is already started and listening'
	ErrServerAlreadyStarted = NewError("Server is already started and listening")
	// ErrServerOptionsMissing returns an error with message: 'You have to pass iris.ServerOptions'
	ErrServerOptionsMissing = NewError("You have to pass iris.ServerOptions")
	// ErrServerTlsOptionsMissing returns an error with message: 'You have to set CertFile and KeyFile to iris.ServerOptions before ListenTLS'
	ErrServerTlsOptionsMissing = NewError("You have to set CertFile and KeyFile to iris.ServerOptions before ListenTLS")
	// ErrServerIsClosed returns an error with message: 'Can't close the server, propably is already closed or never started'
	ErrServerIsClosed = NewError("Can't close the server, propably is already closed or never started")
	// ErrServerUnknown returns an error with message: 'Unknown reason from Server, please report this as bug!'
	ErrServerUnknown = NewError("Unknown reason from Server, please report this as bug!")
	// ErrParsedAddr returns an error with message: 'ListeningAddr error, for TCP and UDP, the syntax of ListeningAddr is host:port, like 127.0.0.1:8080.
	// If host is omitted, as in :8080, Listen listens on all available interfaces instead of just the interface with the given host address.
	// See Dial for more details about address syntax'
	ErrParsedAddr = NewError("ListeningAddr error, for TCP and UDP, the syntax of ListeningAddr is host:port, like 127.0.0.1:8080. If host is omitted, as in :8080, Listen listens on all available interfaces instead of just the interface with the given host address. See Dial for more details about address syntax")

	// Template

	// ErrTemplateParse returns an error with message: 'Couldn't load templates + specific error reason'
	ErrTemplateParse = NewError("Couldn't load templates %s")
	// ErrTemplateWatch returns an error with message: 'Templates watcher couldn't be started, error: + specific error reason'
	ErrTemplateWatch = NewError("Templates watcher couldn't be started, error: %s")
	// ErrTemplateWatching returns an error with message: 'Error while watching templates: + specific error reason'
	ErrTemplateWatching = NewError("While watching templates: %s")

	// Form

	// ErrNoForm returns an error with message: 'Request has no any valid form'
	ErrNoForm = NewError("Request has no any valid form")

	// File

	// ErrNoZip returns an error with message: 'Error while creating +filename. It's not a zip'
	ErrNoZip = NewError("while creating %s. It's not a zip")
)

// This is for tomorrow
type (
	ErrorImpl interface {
		Error() string
		Format(...interface{}) error
		Return() error
	}

	Error struct {
		err error
	}
)

func (e *Error) Error() string {
	return e.err.Error()
}

func (e *Error) Format(args ...interface{}) error {
	return fmt.Errorf(e.err.Error(), args...)
}

func (e *Error) Return() error {
	return e.err
}

//

// NewError creates and returns an error with a message and optionally arguments used in this messsage
func NewError(format string, args ...interface{}) error {
	return fmt.Errorf("\n"+LoggerIrisPrefix+"Error: "+format, args...)
}

// Printf prints to the logger a specific error with optionally arguments
func Printf(logger *Logger, err error, args ...interface{}) {
	logger.Printf(err.Error(), args...)
}
