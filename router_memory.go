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
	"time"
)

type IMemoryRouter interface {
	setCache(IRouterCache)
	hasCache() bool
}

// MemoryRouter is the cached version of the Router
type MemoryRouter struct {
	*Router
	cache         IRouterCache
	maxitems      int
	resetDuration time.Duration
	hasStarted    bool
}

// NewMemoryRouter returns a MemoryRouter
// receives an underline *Router object and int options like MaxItems and ResetDurationTime
func NewMemoryRouter(underlineRouter *Router, maxitems int, resetDuration time.Duration) *MemoryRouter {
	r := &MemoryRouter{}
	r.Router = underlineRouter
	r.maxitems = maxitems
	r.resetDuration = resetDuration
	// CACHE IS CREATED DYNAMICLY BEFORE THE LISTEN ON THE STATION_PREPARE_PLUGIN
	return r
}

func (r *MemoryRouter) setCache(cache IRouterCache) {
	r.cache = cache
	r.cache.SetMaxItems(r.maxitems)
	ticker := NewTicker()
	ticker.OnTick(r.cache.OnTick) // registers the cache to the ticker
	ticker.Start(r.resetDuration) //starts the ticker now
	r.hasStarted = true
}

func (r *MemoryRouter) hasCache() bool {
	return r.cache != nil && r.hasStarted
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
		ctx.Redo(res, req)
		return
	}

	ctx := r.getStation().pool.Get().(*Context)
	ctx.Reset(res, req)

	if r.processRequest(ctx) {
		//if something found and served then add it's clone to the cache
		r.cache.AddItem(method, path, ctx.Clone())
	}

	r.getStation().pool.Put(ctx)

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

func (r *MemoryRouterDomain) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	method := req.Method

	path := req.URL.Path + req.Host
	if ctx := r.cache.GetItem(method, path); ctx != nil {
		ctx.Redo(res, req)
		return
	}

	ctx := r.getStation().pool.Get().(*Context)
	ctx.Reset(res, req)

	if r.processRequest(ctx) {
		//if something found and served then add it's context's clone to the cache
		r.cache.AddItem(method, path, ctx.Clone())
	}

	r.getStation().pool.Put(ctx)
}
