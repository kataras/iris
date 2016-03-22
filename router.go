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
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

import (
	"net/http"
	"net/url"
	"path"
	"strings"
)

const (
	// ParameterStartByte is very used on the node, it's just contains the byte for the ':' rune/char
	ParameterStartByte = byte(':')
	// SlashByte is just a byte of '/' rune/char
	SlashByte = byte('/')
	// MatchEverythingByte is just a byte of '*" rune/char
	MatchEverythingByte = byte('*')
)

type RouterType uint8

const (
	Normal RouterType = iota
	Memory
	NormalDomain
	MemoryDomain
)

// IRouter is the interface of which any Iris router must implement
type IRouter interface {
	IParty
	getGarden() Garden
	setGarden(g Garden)
	getType() RouterType
	getStation() *Station
	Errors() IHTTPErrors //at the main Router struct this is managed by the MiddlewareSupporter
	// ServeHTTP finds and serves a route by it's request
	// If no route found, it sends an http status 404
	ServeHTTP(http.ResponseWriter, *http.Request)
	processRequest(*Context) bool
}

// Router is the router , one router per server.
// Router contains the global middleware, the routes and a Mutex for lock and unlock on route prepare
type Router struct {
	station *Station
	IParty
	garden     Garden
	httpErrors IHTTPErrors //the only reason of this is to pass into the route, which it need it to  passed it to Context, in order to  developer get the ability to perfom emit errors (eg NotFound) directly from context
}

var _ IRouter = &Router{}

// NewRouter creates and returns an empty Router
func NewRouter(station *Station) *Router {
	r := &Router{station: station, httpErrors: DefaultHTTPErrors(), garden: make([]tree, 0, len(HTTPMethods.ANY))} // TODO: maybe +1 for any which is just empty tree ""
	r.IParty = NewParty("/", r.station, nil)
	return r
}
func (r *Router) getGarden() Garden {
	return r.garden
}

func (r *Router) setGarden(g Garden) {
	r.garden = g
} //every plant we make to the garden, garden sets itself

func (r *Router) getType() RouterType {
	return Normal
}

func (r *Router) getStation() *Station {
	return r.station
}

// SetErrors sets a HTTPErrors object to the router
func (r *Router) SetErrors(httperr IHTTPErrors) {
	r.httpErrors = httperr
}

// Errors get the HTTPErrors from the router
func (r *Router) Errors() IHTTPErrors {
	return r.httpErrors
}

func (r *Router) find(_tree tree, reqPath string, ctx *Context) bool {
	middleware, params, mustRedirect := _tree.rootBranch.GetBranch(reqPath, ctx.Params) // pass the parameters here for 0 allocation
	if middleware != nil {
		ctx.Params = params
		ctx.middleware = middleware
		ctx.Do()
		return true
	} else if mustRedirect && r.station.options.PathCorrection {
		pathLen := len(reqPath)
		//first of all checks if it's the index only slash /
		if pathLen <= 1 {
			reqPath = "/"
			//check if the req path ends with slash
		} else if reqPath[pathLen-1] == '/' {
			reqPath = reqPath[:pathLen-1] //remove the last /
		} else {
			//it has path prefix, it doesn't ends with / and it hasn't be found, then just add the slash
			reqPath = reqPath + "/"
		}
		ctx.Request.URL.Path = reqPath
		urlToRedirect := ctx.Request.URL.String()
		if u, err := url.Parse(urlToRedirect); err == nil {

			if u.Scheme == "" && u.Host == "" {
				//The http://yourserver is done automatically by all browsers today
				//so just clean the path
				trailing := strings.HasSuffix(urlToRedirect, "/")
				urlToRedirect = path.Clean(urlToRedirect)
				//check after clean if we had a slash but after we don't, we have to do that otherwise we will get forever redirects if path is /home but the registed is /home/
				if trailing && !strings.HasSuffix(urlToRedirect, "/") {
					urlToRedirect += "/"
				}

			}

			ctx.ResponseWriter.Header().Set("Location", urlToRedirect)
			ctx.ResponseWriter.WriteHeader(http.StatusMovedPermanently)

			// RFC2616 recommends that a short note "SHOULD" be included in the
			// response because older user agents may not understand 301/307.
			// Shouldn't send the response for POST or HEAD; that leaves GET.
			if _tree.method == HTTPMethods.GET {
				note := "<a href=\"" + htmlEscape(urlToRedirect) + "\">Moved Permanently</a>.\n"
				ctx.Write(note)
			}
			return false
		}
	}
	ctx.NotFound()
	return false

}

//we use that to the router_memory also
//returns true if it actually find serve something
func (r *Router) processRequest(ctx *Context) bool {
	reqPath := ctx.Request.URL.Path
	method := ctx.Request.Method
	gLen := len(r.garden)
	for i := 0; i < gLen; i++ {
		if r.garden[i].method == method {
			return r.find(r.garden[i], reqPath, ctx)
		}
	}
	ctx.NotFound()
	return false
}

///////////////////////////////
//expose some methods as public
///////////////////////////////

// ServeHTTP finds and serves a route by it's request
// If no route found, it sends an http status 404
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	ctx := r.station.pool.Get().(*Context)
	ctx.Reset(res, req)

	//defer r.station.pool.Put(ctx)
	// defer is too slow it adds 10k nanoseconds to the benchmarks...so I will wrap the below to a function
	r.processRequest(ctx)

	r.station.pool.Put(ctx)

}

// RouterDomain same as Router but it's override the ServeHTTP and proccessPath.
type RouterDomain struct {
	*Router
}

func NewRouterDomain(underlineRouter *Router) *RouterDomain {
	return &RouterDomain{underlineRouter}
}

func (r RouterDomain) getType() RouterType {
	return NormalDomain
}

func (r *RouterDomain) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	ctx := r.station.pool.Get().(*Context)
	ctx.Reset(res, req)

	//defer r.station.pool.Put(ctx)
	// defer is too slow it adds 10k nanoseconds to the benchmarks...so I will wrap the below to a function
	r.processRequest(ctx)

	r.station.pool.Put(ctx)

}

// all these dublicates for this if: if r.garden[i].hosts { but it's 3k nanoseconds faster on non-domain routers, so I keep it FOR NOW I WILL FIND BETTER WAY
func (r *RouterDomain) processRequest(ctx *Context) bool {
	reqPath := ctx.Request.URL.Path
	gLen := len(r.garden)
	for i := 0; i < gLen; i++ {
		if r.garden[i].hosts {
			//it's expecting host
			if r.garden[i].domain != ctx.Request.Host {
				//but this is not the host we were expecting, so just continue to the next
				continue
			}
			reqPath = ctx.Request.Host + reqPath
		}
		if r.garden[i].method == ctx.Request.Method {

			return r.find(r.garden[i], reqPath, ctx)
		}

	}
	ctx.NotFound()
	return false
}
