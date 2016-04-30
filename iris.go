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
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

//Package iris v2.0.0-alpha
package iris

import (
	"github.com/kataras/iris/errors"
)

const (
	// DefaultProfilePath is the default path for the web pprof '/debug/pprof'
	DefaultProfilePath = "/debug/pprof"

	// Content & Header

	// DefaultUserAgent default to 'iris' but it is not used anywhere yet
	DefaultUserAgent = "iris"
	// DefaultCharset represents the default charset for content headers
	DefaultCharset = "UTF-8"
	// ContentType represents the header["Content-Type"]
	ContentType = "Content-Type"
	// ContentLength represents the header["Content-Length"]
	ContentLength = "Content-Length"
	// ContentHTML is the  string of text/html response headers
	ContentHTML = "text/html"
	// ContentJSON is the  string of application/json response headers
	ContentJSON = "application/json"
	// ContentJSONP is the  string of application/javascript response headers
	ContentJSONP = "application/javascript"
	// ContentBINARY is the  string of "application/octet-stream response headers
	ContentBINARY = "application/octet-stream"
	// ContentTEXT is the  string of text/plain response headers
	ContentTEXT = "text/plain"
	// ContentXML is the  string of application/xml response headers
	ContentXML = "application/xml"
	// ContentXMLText is the  string of text/xml response headers
	ContentXMLText = "text/xml"

	// LastModified "Last-Modified"
	LastModified = "Last-Modified"
	// IfModifiedSince "If-Modified-Since"
	IfModifiedSince = "If-Modified-Since"
	// ContentDisposition "Content-Disposition"
	ContentDisposition = "Content-Disposition"

	// TimeFormat default time format for any kind of datetime parsing
	TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

	// stopExecutionPosition used inside the Context, is the number which shows us that the context's middleware manualy stop the execution
	stopExecutionPosition = 255

	// ParameterStartByte is very used on the node, it's just contains the byte for the ':' rune/char
	ParameterStartByte = byte(':')
	// SlashByte is just a byte of '/' rune/char
	SlashByte = byte('/')
	// Slash is just a string of "/"
	Slash = "/"
	// MatchEverythingByte is just a byte of '*" rune/char
	MatchEverythingByte = byte('*')

	// HTTP Methods(1)

	MethodGet     = "GET"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodHead    = "HEAD"
	MethodPatch   = "PATCH"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
)

var (
	// DefaultStation in order to use iris.Get(...,...) we need a default server on the package level
	DefaultStation *Station

	// Errors

	// Router, Party & Handler

	// ErrHandler returns na error with message: 'Passed argument is not func(*Context) neither an object which implements the iris.Handler with Serve(ctx *Context)
	// It seems to be a  +type Points to: +pointer.'
	ErrHandler = errors.New("Passed argument is not func(*Context) neither an object which implements the iris.Handler with Serve(ctx *Context)\n It seems to be a  %T Points to: %v.")
	// ErrHandleAnnotated returns an error with message: 'HandleAnnotated parse: +specific error(s)'
	ErrHandleAnnotated = errors.New("HandleAnnotated parse: %s")

	// Plugin

	// ErrPluginAlreadyExists returns an error with message: 'Cannot activate the same plugin again, plugin '+plugin name[+plugin description]' is already exists'
	ErrPluginAlreadyExists = errors.New("Cannot use the same plugin again, '%s[%s]' is already exists")
	// ErrPluginActivate returns an error with message: 'While trying to activate plugin '+plugin name'. Trace: +specific error'
	ErrPluginActivate = errors.New("While trying to activate plugin '%s'. Trace: %s")
	// ErrPluginRemoveNoPlugins returns an error with message: 'No plugins are registed yet, you cannot remove a plugin from an empty list!'
	ErrPluginRemoveNoPlugins = errors.New("No plugins are registed yet, you cannot remove a plugin from an empty list!")
	// ErrPluginRemoveEmptyName returns an error with message: 'Plugin with an empty name cannot be removed'
	ErrPluginRemoveEmptyName = errors.New("Plugin with an empty name cannot be removed")
	// ErrPluginRemoveNotFound returns an error with message: 'Cannot remove a plugin which doesn't exists'
	ErrPluginRemoveNotFound = errors.New("Cannot remove a plugin which doesn't exists")
	// Context other

	// ErrNoForm returns an error with message: 'Request has no any valid form'
	ErrNoForm = errors.New("Request has no any valid form")
	// ErrWriteJSON returns an error with message: 'Before JSON be written to the body, JSON Encoder returned an error. Trace: +specific error'
	ErrWriteJSON = errors.New("Before JSON be written to the body, JSON Encoder returned an error. Trace: %s")
	// ErrRenderMarshalled returns an error with message: 'Before +type Rendering, MarshalIndent retured an error. Trace: +specific error'
	ErrRenderMarshalled = errors.New("Before +type Rendering, MarshalIndent returned an error. Trace: %s")
	// ErrReadBody returns an error with message: 'While trying to read +type from the request body. Trace +specific error'
	ErrReadBody = errors.New("While trying to read %s from the request body. Trace %s")
	// ErrServeContent returns an error with message: 'While trying to serve content to the client. Trace +specific error'
	ErrServeContent = errors.New("While trying to serve content to the client. Trace %s")

	// File & Dir

	// Storage
	// ErrFlashNotFound returns an error with message: 'Unable to get flash message. Trace: Cookie does not exists'
	ErrFlashNotFound = errors.New("Unable to get flash message. Trace: Cookie does not exists")

	// HTTP Methods(2)

	MethodConnectBytes = []byte(MethodConnect)

	AllMethods = [...]string{"GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE"}
)

// init the only one.
func init() {
	DefaultStation = New()
}
