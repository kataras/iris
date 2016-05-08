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

// Package iris Context.go  context_binder.go, context_renderer.go, context_storage.go, context_request.go, context_response.go
package iris

import (
	"reflect"
	"runtime"
	"time"

	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
)

const (
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
	// ContentBINARY is the string of application/octet-stream response headers
	ContentBINARY = "application/octet-stream"

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
)

// Charset is defaulted to UTF-8, you can change it
// all render methods will have this charset
var Charset = DefaultCharset

type (

	// IContext the interface for the Context
	IContext interface {
		context.Context
		IContextRenderer
		IContextStorage
		IContextBinder
		IContextRequest
		IContextResponse

		Reset(*fasthttp.RequestCtx)
		Clone() *Context
		Do()
		Next()
		StopExecution()
		IsStopped() bool
		GetHandlerName() string
	}

	// Context is resetting every time a request is coming to the server
	// it is not good practice to use this object in goroutines, for these cases use the .Clone()
	Context struct {
		*fasthttp.RequestCtx
		Params  PathParameters
		station *Iris
		//keep track all registed middleware (handlers)
		middleware Middleware
		// pos is the position number of the Context, look .Next to understand
		pos uint8
		// these values are reseting on each request, are useful only between middleware,
		// use iris/sessions for cookie/filesystem storage
		values map[interface{}]interface{}
	}
)

var _ IContext = &Context{}

// Implement the golang.org/x/net/context , as requested by the community, which is used inside app engine
// also this will give me the ability to use appengine's memcache with this context, if this needed.

// Deadline returns the time when this Context will be canceled, if any.
func (ctx *Context) Deadline() (deadline time.Time, ok bool) {
	return
}

// Done returns a channel that is closed when this Context is canceled
// or times out.
func (ctx *Context) Done() <-chan struct{} {
	return nil
}

// Err indicates why this context was canceled, after the Done channel
// is closed.
func (ctx *Context) Err() error {
	return nil
}

// Value returns the value associated with key or nil if none.
func (ctx *Context) Value(key interface{}) interface{} {
	if key == 0 {
		return ctx.Request
	}
	if keyAsString, ok := key.(string); ok {
		val := ctx.GetString(keyAsString)
		return val
	}
	return nil
}

// Reset resets the Context with a given domain.Response and domain.Request
// the context is ready-to-use after that, just like a new Context
// I use it for zero rellocation memory
func (ctx *Context) Reset(reqCtx *fasthttp.RequestCtx) {
	ctx.Params = ctx.Params[0:0]
	ctx.middleware = nil
	ctx.RequestCtx = reqCtx
}

// Clone use that method if you want to use the context inside a goroutine
func (ctx *Context) Clone() *Context {
	var cloneContext = *ctx
	cloneContext.pos = 0

	//copy params
	p := ctx.Params
	cpP := make(PathParameters, len(p))
	copy(cpP, p)
	cloneContext.Params = cpP
	//copy middleware
	m := ctx.middleware
	cpM := make(Middleware, len(m))
	copy(cpM, m)
	cloneContext.middleware = cpM
	return &cloneContext
}

// Do calls the first handler only, it's like Next with negative pos, used only on Router&MemoryRouter
func (ctx *Context) Do() {
	ctx.pos = 0
	ctx.middleware[0].Serve(ctx)
}

// Next calls all the next handler from the middleware stack, it used inside a middleware
func (ctx *Context) Next() {
	//set position to the next
	ctx.pos++
	midLen := uint8(len(ctx.middleware)) // max 255 handlers, we don't except more than these logically ...
	//run the next
	if ctx.pos < midLen {
		ctx.middleware[ctx.pos].Serve(ctx)
	}

}

// StopExecution just sets the .pos to 255 in order to  not move to the next middlewares(if any)
func (ctx *Context) StopExecution() {
	ctx.pos = stopExecutionPosition
}

//

// IsStopped checks and returns true if the current position of the Context is 255, means that the StopExecution has called
func (ctx *Context) IsStopped() bool {
	return ctx.pos == stopExecutionPosition
}

// GetHandlerName as requested returns the stack-name of the function which the Middleware is setted from
func (ctx *Context) GetHandlerName() string {
	return runtime.FuncForPC(reflect.ValueOf(ctx.middleware[len(ctx.middleware)-1]).Pointer()).Name()
}
