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

type (
	IContextResponse interface {
		// SetContentType sets the "Content-Type" header, receives the values
		SetContentType([]string)
		// SetHeader sets the response headers first parameter is the key, second is the values
		SetHeader(string, []string)
		Redirect(string, ...int)
		// Errors
		NotFound()
		Panic()
		EmitError(int)
		//
	}
)

// SetContentType sets the response writer's header key 'Content-Type' to a given value(s)
func (ctx *Context) SetContentType(s []string) {
	for _, hv := range s {
		ctx.RequestCtx.Response.Header.Set(ContentType, hv)
	}

}

// SetHeader write to the response writer's header to a given key the given value(s)
func (ctx *Context) SetHeader(k string, s []string) {
	for _, hv := range s {
		ctx.RequestCtx.Response.Header.Set(k, hv)
	}
}

// Redirect redirect sends a redirect response the client
// accepts 2 parameters string and an optional int
// first parameter is the url to redirect
// second parameter is the http status should send, default is 302 (Temporary redirect), you can set it to 301 (Permant redirect), if that's nessecery
func (ctx *Context) Redirect(urlToRedirect string, statusHeader ...int) {
	httpStatus := 302 // temporary redirect
	if statusHeader != nil && len(statusHeader) > 0 && statusHeader[0] > 0 {
		httpStatus = statusHeader[0]
	}

	ctx.RequestCtx.Redirect(urlToRedirect, httpStatus)
}

// Error handling

// NotFound emits an error 404 to the client, using the custom http errors
// if no custom errors provided then it sends the default http.NotFound
func (ctx *Context) NotFound() {
	ctx.StopExecution()
	ctx.station.EmitError(404, ctx)
}

// Panic stops the executions of the context and returns the registed panic handler
// or if not, the default which is  500 http status to the client
//
// This function is useful when you use the recovery middleware, which is auto-executing the (custom, registed) 500 internal server error.
func (ctx *Context) Panic() {
	ctx.StopExecution()
	ctx.station.EmitError(500, ctx)
}

// EmitError executes the custom error by the http status code passed to the function
func (ctx *Context) EmitError(statusCode int) {
	ctx.station.EmitError(statusCode, ctx)
}
