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
package iris

const (
	//mime types and headers
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

	//statuses
	StatusOK         = 200
	MethodNotAllowed = 503

	//other

	TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

	// stopExecutionPosition used inside the Context, is the number which shows us that the context's middleware manualy stop the execution
	stopExecutionPosition = 255 // is the biggest uint8

	// LoggerIrisPrefix is the prefix of the logger '[IRIS] '
	LoggerIrisPrefix = "[IRIS] "

	// DefaultServerAddr the default server addr if nothing passed
	DefaultServerAddr = ":8080"
)

var (
	DefaultStation *Station
)

// The one and only init to the whole package
func init() {
	DefaultStation = New()
}

// New creates and returns a new iris Station with default options
func New() *Station {
	defaultOptions := defaultOptions()
	return newStation(defaultOptions)
}

// Custom is used for iris-experienced developers
// creates and returns a new iris Station with custom StationOptions
func Custom(options StationOptions) *Station {

	if options.ProfilePath != "" {
		options.ProfilePath = DefaultProfilePath
	}

	return newStation(options)
}
