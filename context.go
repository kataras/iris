/*
Context.go  Implements: ./context/context.go ,
files: context_renderer.go, context_storage.go, context_request.go, context_response.go
*/

package iris

import (
	"reflect"
	"runtime"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/sessions/store"
	"github.com/valyala/fasthttp"
)

const (
	// DefaultUserAgent default to 'iris' but it is not used anywhere yet
	DefaultUserAgent = "iris"
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

type (
	// Map is just a conversion for a map[string]interface{}
	Map map[string]interface{}
	// Context is resetting every time a request is coming to the server
	// it is not good practice to use this object in goroutines, for these cases use the .Clone()
	Context struct {
		*fasthttp.RequestCtx
		Params  PathParameters
		station *Iris
		//keep track all registed middleware (handlers)
		middleware   Middleware
		sessionStore store.IStore
		// pos is the position number of the Context, look .Next to understand
		pos uint8
	}
)

var _ context.IContext = &Context{}

// GetRequestCtx returns the current fasthttp context
func (ctx *Context) GetRequestCtx() *fasthttp.RequestCtx {
	return ctx.RequestCtx
}

// Reset resets the Context with a given domain.Response and domain.Request
// the context is ready-to-use after that, just like a new Context
// I use it for zero rellocation memory
func (ctx *Context) Reset(reqCtx *fasthttp.RequestCtx) {
	ctx.Params = ctx.Params[0:0]
	ctx.sessionStore = nil
	ctx.middleware = nil
	ctx.RequestCtx = reqCtx
}

// Clone use that method if you want to use the context inside a goroutine
func (ctx *Context) Clone() context.IContext {
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

	// we don't copy the sessionStore for more than one reasons...
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
	midLen := uint8(len(ctx.middleware))
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
