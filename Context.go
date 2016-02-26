package iris

import (
	"net/http"
	"reflect"
	"strconv"
)

var contextType reflect.Type

// Context is created every time a request is coming to the server,
// it holds a pointer to the http.Request, the ResponseWriter
// and the Named Parameters (if any) of the requested path.
//
// Context is transfering to the frontend dev via the handler,
// from the route.go 's Prepare -> convert handler as middleware and use route.run -> ServeHTTP.
type Context struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Params         Parameters
	errorHandlers  ErrorHandlers
}

// newContext creates and returns a new Context pointer
func newContext(res http.ResponseWriter, req *http.Request, errorHandlers ErrorHandlers) *Context {
	params := Params(req)
	return &Context{ResponseWriter: res, Request: req, Params: params, errorHandlers: errorHandlers}
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
	ctx.ResponseWriter.Write([]byte(contents))
}

// ServeFile is used to serve a file, via the http.ServeFile
func (ctx *Context) ServeFile(path string) {
	http.ServeFile(ctx.ResponseWriter, ctx.Request, path)
}

// GetCookie get cookie's value by it's name
func (ctx *Context) GetCookie(name string) string {
	_cookie, _err := ctx.Request.Cookie(CookieName)
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
	ctx.errorHandlers[http.StatusNotFound].ServeHTTP(ctx.ResponseWriter, ctx.Request)
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
