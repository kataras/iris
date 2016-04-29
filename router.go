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
	"sync"

	"github.com/valyala/fasthttp"
)

const (
	// ParameterStartByte is very used on the node, it's just contains the byte for the ':' rune/char
	ParameterStartByte = byte(':')
	// SlashByte is just a byte of '/' rune/char
	SlashByte = byte('/')
	// Slash is just a string of "/"
	Slash = "/"
	// MatchEverythingByte is just a byte of '*" rune/char
	MatchEverythingByte = byte('*')

	// Normal is the Router
	Normal RouterType = iota
	// Domain is a router which accepts more than one host aka subdomain
	Domain
)

const ()

// DefaultUserAgent default to 'iris' but it is not used anywhere yet
var DefaultUserAgent = []byte("iris")

type (
	// RouterType is just the type which the Router uses to indentify what type is (Normal,Memory,MemorySync,Domain,DomainMemory )
	RouterType uint8

	// IRouter is the interface of which any Iris router must implement
	IRouter interface {
		IParty
		RequestHandler
		getGarden() *Garden
		setGarden(g *Garden)
		getType() RouterType
		getStation() *Station
		// Errors
		Errors() IHTTPErrors
		OnError(int, HandlerFunc)
		// EmitError emits an error with it's http status code and the iris Context passed to the function
		EmitError(int, *Context)
		// OnNotFound sets the handler for http status 404,
		// default is a response with text: 'Not Found' and status: 404
		OnNotFound(HandlerFunc)
		// OnPanic sets the handler for http status 500,
		// default is a response with text: The server encountered an unexpected condition which prevented it from fulfilling the request. and status: 500
		OnPanic(HandlerFunc)
		// Static serves a directory
		// accepts three parameters
		// first parameter is the request url path (string)
		// second parameter is the system directory (string)
		// third parameter is the level (int) of stripSlashes
		// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
		// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
		// * stripSlashes = 2, original path: "/foo/bar", result: ""
		Static(string, string, int)
		setMethodMatch(func(string, string) bool)
	}

	// Router is the router , one router per server.
	// Router contains the global middleware, the routes and a Mutex for lock and unlock on route prepare
	Router struct {
		station    *Station
		httpErrors *HTTPErrors
		IParty
		garden      *Garden
		methodMatch func(m1, m2 string) bool
		// errorPool is responsible to  get the Context to handle not found errors
		errorPool sync.Pool
	}
)

var _ IRouter = &Router{}

// CorsMethodMatch is sets the methodMatch when cors enabled (look OptimusPrime), it's allowing OPTIONS method to all other methods except GET
//just this
func CorsMethodMatch(m1, reqMethod string) bool {
	return m1 == reqMethod || (m1 != HTTPMethods.Get && reqMethod == HTTPMethods.Options)
}

// MethodMatch for normal method match
func MethodMatch(m1, m2 string) bool {
	return m1 == m2
}

// NewRouter creates and returns an empty Router
func NewRouter(station *Station) *Router {
	r := &Router{station: station, httpErrors: defaultHTTPErrors(), garden: &Garden{station: station}} // TODO: maybe +1 for any which is just empty tree ""
	r.methodMatch = MethodMatch
	r.IParty = &GardenParty{relativePath: "/", station: r.station, root: true}
	r.errorPool = sync.Pool{New: func() interface{} {
		return &Context{station: station}
	}}
	return r
}

func (r *Router) getGarden() *Garden {
	return r.garden
}

func (r *Router) setGarden(g *Garden) {
	r.garden = g
} //every plant we make to the garden, garden sets itself

func (r *Router) getType() RouterType {
	return Normal
}

func (r *Router) getStation() *Station {
	return r.station
}

func (r *Router) setMethodMatch(f func(m1, m2 string) bool) {
	r.methodMatch = f
}

// Error handling

// Errors returns the object which is resposible for the error(s) handler(s)
func (r *Router) Errors() IHTTPErrors {
	return r.httpErrors
}

// OnError registers a handler ( type of HandlerFunc) for a specific http error status
func (r *Router) OnError(statusCode int, handlerFunc HandlerFunc) {
	r.httpErrors.On(statusCode, handlerFunc)
}

// EmitError emits an error with it's http status code and the iris Context passed to the function
func (r *Router) EmitError(statusCode int, ctx *Context) {
	r.httpErrors.Emit(statusCode, ctx)
}

// OnNotFound sets the handler for http status 404,
// default is a response with text: 'Not Found' and status: 404
func (r *Router) OnNotFound(handlerFunc HandlerFunc) {
	r.OnError(http.StatusNotFound, handlerFunc)
}

// OnPanic sets the handler for http status 500,
// default is a response with text: The server encountered an unexpected condition which prevented it from fulfilling the request. and status: 500
func (r *Router) OnPanic(handlerFunc HandlerFunc) {
	r.OnError(http.StatusInternalServerError, handlerFunc)
}

///////////////////////////////
//expose some methods as public
///////////////////////////////

// Static registers a route which serves a system directory
func (r *Router) Static(requestPath string, systemPath string, stripSlashes int) {
	h := fasthttp.FSHandler(systemPath, stripSlashes)
	r.Get(requestPath+"/*filepath", func(c *Context) {
		h(c.RequestCtx)
	})
}

// ServeRequest finds and serves a route by it's request context
// If no route found, it sends an http status 404
func (r *Router) ServeRequest(reqCtx *fasthttp.RequestCtx) {
	method := BytesToString(reqCtx.Method())
	tree := r.garden.first
	path := BytesToString(reqCtx.Path())
	for tree != nil {
		if r.methodMatch(tree.method, method) {
			if !tree.serve(reqCtx, path) {
				r.notFoundCtx(reqCtx)
			}
			return
		}
		tree = tree.next
	}
	//not found, get the first's pool and use that  to send a custom http error(if setted)

	r.notFoundCtx(reqCtx)

}

// internal method, it justs takes the context from pool ( in order to have the custom errors available) and procedure a Not Found 404 error
// this is being called when no route was found used on the ServeRequest.
func (r *Router) notFoundCtx(reqCtx *fasthttp.RequestCtx) {
	ctx := r.errorPool.Get().(*Context)
	ctx.Reset(reqCtx)
	ctx.NotFound()
	r.errorPool.Put(ctx)
}

// RouterDomain same as Router but it's override the ServeHTTP and processPath.
type RouterDomain struct {
	*Router
}

// NewRouterDomain creates a RouterDomain from an underline (normal) Router and returns it
func NewRouterDomain(underlineRouter *Router) *RouterDomain {
	return &RouterDomain{underlineRouter}
}

func (r *RouterDomain) getType() RouterType {
	return Domain
}

// ServeRequest finds and serves a route by it's request context
// If no route found, it sends an http status 404
func (r *RouterDomain) ServeRequest(reqCtx *fasthttp.RequestCtx) {

	method := BytesToString(reqCtx.Method())
	domain := BytesToString(reqCtx.Host())
	path := reqCtx.Path()
	tree := r.garden.first
	for tree != nil {
		if tree.hosts && tree.domain == domain {
			// here we have an issue, at fasthttp/uri.go 273-274 line normalize path it adds a '/' slash at the beggining, it doesn't checks for subdomains
			// I could fix it but i leave it as it is, I just create a new function inside tree named 'serveReturn' which accepts a path too. ->
			//-> reqCtx.Request.URI().SetPathBytes(append(reqCtx.Host(), reqCtx.Path()...)) <-
			path = append(reqCtx.Host(), reqCtx.Path()...)
		}
		if r.methodMatch(tree.method, method) {
			if tree.serve(reqCtx, BytesToString(path)) {
				return
			}
		}
		tree = tree.next
	}
	//not found, get the first's pool and use that  to send a custom http error(if setted)
	r.notFoundCtx(reqCtx)
}
