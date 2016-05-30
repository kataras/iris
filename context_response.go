package iris

// SetContentType sets the response writer's header key 'Content-Type' to a given value(s)
func (ctx *Context) SetContentType(s string) {
	ctx.RequestCtx.Response.Header.Set(ContentType, s)
}

// SetHeader write to the response writer's header to a given key the given value(s)
func (ctx *Context) SetHeader(k string, v string) {
	ctx.RequestCtx.Response.Header.Set(k, v)
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
	ctx.StopExecution()
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
