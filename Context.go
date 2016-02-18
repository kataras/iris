package iris

import (
	"net/http"
	"reflect"
	"strconv"
)

var contextType reflect.Type

/*
Context is created every time a request is coming to the server,
it holds a pointer to the http.Request, the ResponseWriter
and the Named Parameters (if any) of the requested path.

Context is transfering to the frontend dev via the handler,
from the route.go 's Prepare -> convert handler as middleware and use route.run -> ServeHTTP.
*/
type Context struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Params         Parameters
}

//NewContext creates and returns a new Context pointer
func NewContext(res http.ResponseWriter, req *http.Request) *Context {
	params := Params(req)
	return &Context{ResponseWriter: res, Request: req, Params: params}
}

//Param returns the string representation of the key's value
func (ctx *Context) Param(key string) string {
	return ctx.Params.Get(key)
}

//ParamInt returns the int representation of the key's value
func (ctx *Context) ParamInt(key string) (int, error) {
	val, err := strconv.Atoi(ctx.Params.Get(key))
	return val, err
}

//Write writes a string via the context's ResponseWriter
func (ctx *Context) Write(contents string) {
	ctx.ResponseWriter.Write([]byte(contents))
}

//NotFound sends a http.NotFound response to the client
func (ctx *Context) NotFound() {
	http.NotFound(ctx.ResponseWriter, ctx.Request)
}

//Close is used to close the body of the request
///TODO: CHECK FOR REQUEST CLOSED IN ORDER TO FIX SOME ERRORS HERE
func (ctx *Context) Close() {
	ctx.Request.Body.Close()
}

//ServeFile is used to serve a file, via the http.ServeFile
func (ctx *Context) ServeFile(path string) {
	http.ServeFile(ctx.ResponseWriter, ctx.Request, path)
}
