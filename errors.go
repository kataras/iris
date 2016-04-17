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
package iris

import (
	"fmt"
)

var (
	// Server
	ErrServerPortAlreadyUsed   = NewError("Server can't run, port is already used")
	ErrServerAlreadyStarted    = NewError("Server is already started and listening")
	ErrServerOptionsMissing    = NewError("You have to pass iris.ServerOptions")
	ErrServerTlsOptionsMissing = NewError("You have to set CertFile and KeyFile to iris.ServerOptions before ListenTLS")
	ErrServerIsClosed          = NewError("Can't close the server, propably is already closed or never started")
	ErrServerUnknown           = NewError("Unknown error from Server")
	ErrParsedAddr              = NewError("ListeningAddr error, for TCP and UDP, the syntax of ListeningAddr is host:port, like 127.0.0.1:8080. If host is omitted, as in :8080, Listen listens on all available interfaces instead of just the interface with the given host address. See Dial for more details about address syntax")

	// Template
	ErrTemplateParse    = NewError("Couldn't load templates %s")
	ErrTemplateWatch    = NewError("Templates watcher couldn't be started, error: %s")
	ErrTemplateWatching = NewError("Error when watching templates: %s")
)

func NewError(format string, args ...interface{}) error {
	return fmt.Errorf(LoggerIrisPrefix+"Error: "+format, args...)
}

func Printf(logger *Logger, err error, args ...interface{}) {
	logger.Printf(err.Error(), args...)
}
