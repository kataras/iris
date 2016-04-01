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

import (
	"fmt"
	"net/http"
)

// Handler the main Iris Handler interface.
type Handler interface {
	Serve(ctx *Context)
}

// HandlerFunc type is an adapter to allow the use of
// ordinary functions as HTTP handlers.  If f is a function
// with the appropriate signature, HandlerFunc(f) is a
// Handler that calls f.
type HandlerFunc func(*Context)

// Serve serves the handler, is like ServeHTTP for Iris
func (h HandlerFunc) Serve(ctx *Context) {
	h(ctx)
}

//IMiddlewareSupporter is an interface which all routers must implement
type IMiddlewareSupporter interface {
	Use(handlers ...Handler)
	UseFunc(handlersFn ...HandlerFunc)
}

// Middleware is just a slice of Handler []func(c *Context)
type Middleware []Handler

//MiddlewareSupporter is the struch which make the Imiddlewaresupporter's works, is useful only to no repeat the code of middleware
type MiddlewareSupporter struct {
	Middleware Middleware
}

// Static is just a function which returns a HandlerFunc with the standar http's fileserver's handler
// It is not a middleware, it just returns a HandlerFunc to use anywhere we want
func Static(SystemPath string, PathToStrip ...string) HandlerFunc {
	//runs only once to start the file server
	path := http.Dir(SystemPath)
	underlineFileserver := http.FileServer(path)
	if PathToStrip != nil && len(PathToStrip) == 1 {
		underlineFileserver = http.StripPrefix(PathToStrip[0], underlineFileserver)
	}

	return ToHandlerFunc(underlineFileserver.ServeHTTP)

}

// ToHandler converts http.Handler or func(http.ResponseWriter, *http.Request) to an iris.Handler
func ToHandler(handler interface{}) Handler {
	switch handler.(type) {
	case Handler:
		return handler.(Handler)
	case http.Handler:
		return HandlerFunc((func(ctx *Context) {
			handler.(http.Handler).ServeHTTP(ctx.GetResponseWriter(), ctx.GetRequest())
		}))

	case func(http.ResponseWriter, *http.Request):
		return HandlerFunc((func(ctx *Context) {
			handler.(func(http.ResponseWriter, *http.Request))(ctx.GetResponseWriter(), ctx.GetRequest())
		}))
	default:
		panic(fmt.Sprintf("Error on Iris: handler is not func(*Context) either an object which implements the iris.Handler with  func Serve(ctx *Context)\n It seems to be a  %T Point to: %v:", handler, handler))
	}
}

// ToHandlerFunc converts http.Handler or func(http.ResponseWriter, *http.Request) to an iris.HandlerFunc func (ctx *Context)
func ToHandlerFunc(handler interface{}) HandlerFunc {
	return ToHandler(handler).Serve
}

// ConvertToHandlers accepts list of HandlerFunc and returns list of Handler
// this can be renamed to convertToMiddleware also because it returns a list of []Handler which is what Middleware is
func ConvertToHandlers(handlersFn []HandlerFunc) []Handler {
	hlen := len(handlersFn)
	mlist := make([]Handler, hlen)
	for i := 0; i < hlen; i++ {
		mlist[i] = Handler(handlersFn[i])
	}
	return mlist
}

// JoinMiddleware uses to create a copy of all middleware and return them in order to use inside the node
func JoinMiddleware(middleware1 Middleware, middleware2 Middleware) Middleware {
	nowLen := len(middleware1)
	totalLen := nowLen + len(middleware2)
	// create a new slice of middleware in order to store all handlers, the already handlers(middleware) and the new
	newMiddleware := make(Middleware, totalLen)
	//copy the already middleware to the just created
	copy(newMiddleware, middleware1)
	//start from there we finish, and store the new middleware too
	copy(newMiddleware[nowLen:], middleware2)
	return newMiddleware
}

// Use appends handler(s) to the route or to the router if it's called from router
func (m *MiddlewareSupporter) Use(handlers ...Handler) {
	m.Middleware = append(m.Middleware, handlers...)
	//care here the new handlers will be added to the last, so run Use first for handlers you want to run first
}

// UseFunc is the same as Use but it receives HandlerFunc instead of iris.Handler as parameter(s)
// form of acceptable: func(c *iris.Context){//first middleware}, func(c *iris.Context){//second middleware}
func (m *MiddlewareSupporter) UseFunc(handlersFn ...HandlerFunc) {
	for _, h := range handlersFn {
		m.Use(Handler(h))
	}
}

var _ IMiddlewareSupporter = &MiddlewareSupporter{}
