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
	"net/http"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

type (

	// Handler the main Iris Handler interface.
	Handler interface {
		Serve(ctx *Context)
	}

	// HandlerFunc type is an adapter to allow the use of
	// ordinary functions as HTTP handlers.  If f is a function
	// with the appropriate signature, HandlerFunc(f) is a
	// Handler that calls f.
	HandlerFunc func(*Context)

	//IMiddlewareSupporter is an interface which all routers must implement
	IMiddlewareSupporter interface {
		Use(handlers ...Handler)
		UseFunc(handlersFn ...HandlerFunc)
	}

	// Middleware is just a slice of Handler []func(c *Context)
	Middleware []Handler
)

// Serve serves the handler, is like ServeHTTP for Iris
func (h HandlerFunc) Serve(ctx *Context) {
	h(ctx)
}

// ToHandler converts an http.Handler or http.HandlerFunc to an iris.Handler
func ToHandler(handler interface{}) Handler {
	//this is not the best way to do it, but I dont have any options right now.
	switch handler.(type) {
	case Handler:
		//it's already an iris handler
		return handler.(Handler)
	case http.Handler:
		//it's http.Handler
		h := fasthttpadaptor.NewFastHTTPHandlerFunc(handler.(http.Handler).ServeHTTP)

		return ToHandlerFastHTTP(h)
	case func(http.ResponseWriter, *http.Request):
		//it's http.HandlerFunc
		h := fasthttpadaptor.NewFastHTTPHandlerFunc(handler.(func(http.ResponseWriter, *http.Request)))
		return ToHandlerFastHTTP(h)
	default:
		panic(ErrHandler.Format(handler, handler))
	}
}

// ToHandlerFunc converts an http.Handler or http.HandlerFunc to an iris.HandlerFunc
func ToHandlerFunc(handler interface{}) HandlerFunc {
	return ToHandler(handler).Serve
}

// ToHandlerFastHTTP converts an fasthttp.RequestHandler to an iris.Handler
func ToHandlerFastHTTP(h fasthttp.RequestHandler) Handler {
	return HandlerFunc((func(ctx *Context) {
		h(ctx.RequestCtx)
	}))
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
