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
	// ErrServerRemoveUnix returns an error with message: 'Unexpected error when trying to remove unix socket file +filename: +specific error"'
	ErrServerRemoveUnix = NewError("Unexpected error when trying to remove unix socket file %s: %s")
	// ErrServerChmod returns an error with message: 'Cannot chmod +mode for +host:+specific error
	ErrServerChmod = NewError("Cannot chmod %#o for %q: %s")

	// Router, Party & Handler

	// ErrHandler returns na error with message: 'Passed argument is not func(*Context) neither an object which implements the iris.Handler with Serve(ctx *Context)
	// It seems to be a  +type Points to: +pointer.'
	ErrHandler = NewError("Passed argument is not func(*Context) neither an object which implements the iris.Handler with Serve(ctx *Context)\n It seems to be a  %T Points to: %v.")
	// ErrHandleAnnotated returns an error with message: 'HandleAnnotated parse: +specific error(s)'
	ErrHandleAnnotated = NewError("HandleAnnotated parse: %s")

	// Template

	// ErrTemplateParse returns an error with message: 'Couldn't load templates +specific error'
	ErrTemplateParse = NewError("Couldn't load templates %s")
	// ErrTemplateWatch returns an error with message: 'Templates watcher couldn't be started, error: +specific error'
	ErrTemplateWatch = NewError("Templates watcher couldn't be started, error: %s")
	// ErrTemplateWatching returns an error with message: 'While watching templates: +specific error'
	ErrTemplateWatching = NewError("While watching templates: %s")
	// ErrTemplateExecute returns an error with message:'Unable to execute a template. Trace: +specific error'
	ErrTemplateExecute = NewError("Unable to execute a template. Trace: %s")

	// Plugin

	// ErrPluginAlreadyExists returns an error with message: 'Cannot activate the same plugin again, plugin '+plugin name[+plugin description]' is already exists'
	ErrPluginAlreadyExists = NewError("Cannot use the same plugin again, '%s[%s]' is already exists")
	// ErrPluginActivate returns an error with message: 'While trying to activate plugin '+plugin name'. Trace: +specific error'
	ErrPluginActivate = NewError("While trying to activate plugin '%s'. Trace: %s")

	// Context other

	// ErrNoForm returns an error with message: 'Request has no any valid form'
	ErrNoForm = NewError("Request has no any valid form")
	// ErrWriteJSON returns an error with message: 'Before JSON be written to the body, JSON Encoder returned an error. Trace: +specific error'
	ErrWriteJSON = NewError("Before JSON be written to the body, JSON Encoder returned an error. Trace: %s")
	// ErrRenderMarshalled returns an error with message: 'Before +type Rendering, MarshalIndent retured an error. Trace: +specific error'
	ErrRenderMarshalled = NewError("Before +type Rendering, MarshalIndent returned an error. Trace: %s")
	// ErrReadBody returns an error with message: 'While trying to read +type from the request body. Trace +specific error'
	ErrReadBody = NewError("While trying to read %s from the request body. Trace %s")
	// ErrServeContent returns an error with message: 'While trying to serve content to the client. Trace +specific error'
	ErrServeContent = NewError("While trying to serve content to the client. Trace %s")

	// File & Dir

	// ErrNoZip returns an error with message: 'While creating file '+filename'. It's not a zip'
	ErrNoZip = NewError("While installing file '%s'. It's not a zip")
	// ErrFileOpen returns an error with message: 'While opening a file. Trace: +specific error'
	ErrFileOpen = NewError("While opening a file. Trace: %s")
	// ErrFileCreate returns an error with message: 'While creating a file. Trace: +specific error'
	ErrFileCreate = NewError("While creating a file. Trace: %s")
	// ErrFileRemove returns an error with message: 'While removing a file. Trace: +specific error'
	ErrFileRemove = NewError("While removing a file. Trace: %s")
	// ErrFileCopy returns an error with message: 'While copying files. Trace: +specific error'
	ErrFileCopy = NewError("While copying files. Trace: %s")
	// ErrFileDownload returns an error with message: 'While downloading from +specific url. Trace: +specific error'
	ErrFileDownload = NewError("While downloading from %s. Trace: %s")
	// ErrDirCreate returns an error with message: 'Unable to create directory on '+root dir'. Trace: +specific error
	ErrDirCreate = NewError("Unable to create directory on '%s'. Trace: %s")
)

type Error struct {
	err error
}

func (e *Error) Error() string {
	return e.err.Error()
}

func (e *Error) Format(args ...interface{}) error {
	return fmt.Errorf(e.err.Error(), args...)
}

// With does the same thing as Format but it receives an error type which if it's nil it returns a nil error
func (e *Error) With(err error) error {
	if err == nil {
		return nil
	}

	return e.Format(err.Error())
}

func (e *Error) Return() error {
	return e.err
}

//

// NewError creates and returns an Error with a message
func NewError(errMsg string) *Error {
	return &Error{fmt.Errorf("\n" + LoggerIrisPrefix + "Error: " + errMsg)}
}

// Printf prints to the logger a specific error with optionally arguments
func Printf(logger *Logger, err error, args ...interface{}) {
	logger.Printf(err.Error(), args...)
}
