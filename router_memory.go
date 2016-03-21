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
	"time"
)

// MemoryRouter is the cached version of the Router
type MemoryRouter struct {
	*Router
	cache *MemoryRouterCache
}

// NewMemoryRouter returns a MemoryRouter
// receives an underline *Router object and int options like MaxItems and ResetDurationTime
func NewMemoryRouter(underlineRouter *Router, maxitems int, resetDuration time.Duration) *MemoryRouter {
	r := &MemoryRouter{}
	r.Router = underlineRouter
	//moved to the station.go r.Router = NewRouter(station) // extends all methods from the standar router
	r.cache = NewMemoryRouterCache()
	r.cache.SetMaxItems(maxitems) //no max items just clear every 5 minutes
	ticker := NewTicker()
	ticker.OnTick(r.cache.OnTick) // registers the cache to the ticker
	ticker.Start(resetDuration)   //starts the ticker now

	return r
}
func (r MemoryRouter) getType() RouterType {
	return Memory
}

// ServeHTTP calls processRequest which finds and serves a route by it's request
// If no route found, it sends an http status 404 with a custom error middleware, if setted
func (r *MemoryRouter) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	//16/03/2016 Tried to get/pass only middlewares but it slow me 8k nanoseconds, so I re-do it as I had before.
	method := req.Method

	path := req.URL.Path
	if ctx := r.cache.GetItem(method, path); ctx != nil {
		ctx.Redo(req, res)
		return
	}

	ctx := r.station.pool.Get().(*Context)
	ctx.memoryResponseWriter.New(res)
	ctx.Request = req
	ctx.New()

	if r.processRequest(ctx) {
		//if something found and served then add it's clone to the cache
		r.cache.AddItem(method, path, ctx.Clone())
	}

	r.station.pool.Put(ctx)
}

type MemoryRouterDomain struct {
	*MemoryRouter
}

func NewMemoryRouterDomain(underlineRouter *MemoryRouter) *MemoryRouterDomain {
	return &MemoryRouterDomain{underlineRouter}
}

func (r MemoryRouterDomain) getType() RouterType {
	return MemoryDomain
}
func (r *MemoryRouterDomain) processRequest(ctx *Context) bool {
	reqPath := ctx.Request.URL.Path
	method := ctx.Request.Method
	gLen := len(r.garden)
	for i := 0; i < gLen; i++ {
		if r.garden[i].hosts {
			//it's expecting host
			if r.garden[i].domain != ctx.Request.Host {
				//but this is not the host we are waiting, so just continue
				continue
			}
			reqPath = ctx.Request.Host + reqPath
		}

		if r.garden[i].method == method {
			middleware, params, mustRedirect := r.garden[i].rootBranch.GetBranch(reqPath, ctx.Params) // pass the parameters here for 0 allocation
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
					if method == HTTPMethods.GET {
						note := "<a href=\"" + htmlEscape(urlToRedirect) + "\">Moved Permanently</a>.\n"
						ctx.Write(note)
					}
					return false
				}
			}
		}

	}
	ctx.NotFound()
	return false
}

func (r *MemoryRouterDomain) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	//16/03/2016 Tried to get/pass only middlewares but it slow me 8k nanoseconds, so I re-do it as I had before.
	method := req.Method

	path := req.URL.Path + req.Host //5k nanoseconds down from this oparation
	//change at 21/03/2016, now we are caching all routes either they are attached to the domain or no, with their host all of them.

	if ctx := r.cache.GetItem(method, path); ctx != nil {
		ctx.Redo(req, res)
		return
	}

	ctx := r.station.pool.Get().(*Context)
	ctx.memoryResponseWriter.New(res)
	ctx.Request = req
	ctx.New()

	if r.processRequest(ctx) {
		//if something found and served then add it's clone to the cache
		r.cache.AddItem(method, path, ctx.Clone())
	}

	r.station.pool.Put(ctx)
}
