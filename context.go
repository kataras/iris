package iris

import (
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// Context is created every time a request is coming to the server,
// it holds a pointer to the http.Request, the ResponseWriter
// the Named Parameters (if any) of the requested path and an underline Renderer.
//
// Context is transferring to the frontend dev via the ContextedHandlerFunc at the handler.go,
// from the route.go 's Prepare -> convert handler as middleware and use route.run -> ServeHTTP.
type Context struct {
	*Renderer
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Params         PathParameters
	station        *Station
	//keep track all registed middleware (handlers)
	middleware Middleware
	// pos is the position number of the Context, look .Next to understand
	pos uint8
}

// Param returns the string representation of the key's path named parameter's value
func (ctx *Context) Param(key string) string {
	return ctx.Params.Get(key)
}

// ParamInt returns the int representation of the key's path named parameter's value
func (ctx *Context) ParamInt(key string) (int, error) {
	val, err := strconv.Atoi(ctx.Params.Get(key))
	return val, err
}

// URLParam returns the get parameter from a request , if any
func (ctx *Context) URLParam(key string) string {
	return URLParam(ctx.Request, key)
}

// URLParamInt returns the get parameter int value from a request , if any
func (ctx *Context) URLParamInt(key string) (int, error) {
	return strconv.Atoi(URLParam(ctx.Request, key))
}

// Write writes a string via the context's ResponseWriter
func (ctx *Context) Write(contents string) {
	io.WriteString(ctx.ResponseWriter, contents)
}

// ServeFile is used to serve a file, via the http.ServeFile
func (ctx *Context) ServeFile(path string) {
	http.ServeFile(ctx.ResponseWriter, ctx.Request, path)
}

// GetCookie returns cookie's value by it's name
func (ctx *Context) GetCookie(name string) string {
	//thanks to  @wsantos fix cookieName to name
	_cookie, _err := ctx.Request.Cookie(name)
	if _err != nil {
		return ""
	}
	return _cookie.Value
}

// SetCookie adds a cookie to the request
func (ctx *Context) SetCookie(name string, value string) {
	c := &http.Cookie{Name: name, Value: value}
	ctx.Request.AddCookie(c)
}

// I though about to do it at the Renderer struct, but I think it is better to have the Renderer struct only for
// bigger things, because the word Render does not mean just write, but here in context we have a 'low level' write operators (?)
// I will do it like that, and we'll see

// NotFound emits an error 404 to the client, using the custom http errors
// if no custom errors provided then use the default http.NotFound
// which is already registed nothing special to do here
func (ctx *Context) NotFound() {
	ctx.station.Errors().NotFound(ctx.ResponseWriter)
}

// RequestIP gets just the Remote Address from the client.
func (ctx *Context) RequestIP() string {
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(ctx.Request.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

// Close is used to close the body of the request
///TODO: CHECK FOR REQUEST CLOSED IN ORDER TO FIX SOME ERRORS HERE
func (ctx *Context) Close() {
	ctx.Request.Body.Close()
}

// End same as Close, end the response process.
func (ctx *Context) End() {
	ctx.Request.Body.Close()
}

// Clone before we had (c Context) inscope and  (c *Context) for outscope like goroutines
// now we have (c *Context) for both sittuations ,and call .Clone() if we need to pass the context in a gorotoune or to a time func
// example:
// api.Get("/user/:id", func(ctx *iris.Context) {
//		c:= ctx.Clone()
//		time.AfterFunc(20 * time.Second, func() {
//			println(" 20 secs after: from user with id:", c.Param("id"), " context req path:", c.Request.URL.Path)
//		})
//	})
func (ctx *Context) Clone() *Context {
	cloneContext := *ctx
	//copy params
	params := cloneContext.Params
	cpP := make(PathParameters, len(params))
	copy(cpP, params)
	//copy middleware stack
	//cpM := make(Middleware, len(ctx.middleware))
	//copy(cpM, cloneContext.middleware)

	return &cloneContext
}

// Do calls all the handlers from the middleware stack, it used inside a middleware and on the router's ServeHTTP ( there we use .Do but it's the same).
func (ctx *Context) Do() {
	midLen := uint8(len(ctx.middleware)) // max 255 handlers, we don't except more than these logically ...
	//run all remeaning handlers with this context
	for ctx.pos < midLen {

		ctx.middleware[ctx.pos].Serve(ctx)
		//step to the next
		ctx.pos++
	}
}

func (ctx *Context) clear() {
	ctx.Params = ctx.Params[0:0]
	ctx.ResponseWriter = nil
	ctx.Renderer.responseWriter = nil
	ctx.Request = nil
	ctx.middleware = nil
}
