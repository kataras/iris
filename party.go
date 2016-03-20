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
	"strings"
)

// IPartyHoster is the interface which implements the Party func
type IPartyHoster interface {
	Party(path string) IParty
}

// IParty is the interface which implements the whole Party of routes
type IParty interface {
	IRouterMethods
	IPartyHoster
	IMiddlewareSupporter
	SetParentHosterMiddleware(m Middleware)
	// Each party can have a party too
}

// maybe at the future this will be by-default to all routes, and no need to handle it at different struct
// but this is unnecessary because both nodesprefix and tree are auto-sorted on the tree struct.
// so we will have the main router which is the router.go and all logic implementation is there,
// we have the router_memory which is just a IRouter and has underline router the Router also
// and the route_party which exists on both router and router_memory ofcourse

// party is used inside Router.Party method
type party struct {
	IParty
	middleware Middleware
	_router    IRouter
	_routes    []IRoute // contains all the temporary routes for this party, it is used only from the .Use and .UseFunc to find pathprefixes
	_rootPath  string
}

func NewParty(rootPath string, underlineMainRouter IRouter) IParty {
	p := party{}
	p._router = underlineMainRouter

	//we don't want the root path ends with /
	lastSlashIndex := strings.LastIndexByte(rootPath, SlashByte)

	if lastSlashIndex == len(rootPath)-1 {
		rootPath = rootPath[0:lastSlashIndex]
	}

	p._rootPath = rootPath
	return p
}

//prepared returns a prepared route, just append to the route's the party's middleware([]Handler)
func (p party) prepared(_route IRoute) *Route {
	if len(p.middleware) > 0 {
		//swap them, the party's handlers go first ofc...
		_route.SetMiddleware(append(p.middleware, _route.GetMiddleware()...))
	}
	return _route.(*Route)
}

func (p party) Get(path string, handlerFn ...HandlerFunc) IRoute {
	return p.prepared(p._router.Get(p._rootPath+path, handlerFn...))
}
func (p party) Post(path string, handlerFn ...HandlerFunc) IRoute {
	return p.prepared(p._router.Post(p._rootPath+path, handlerFn...))
}
func (p party) Put(path string, handlerFn ...HandlerFunc) IRoute {
	return p.prepared(p._router.Put(p._rootPath+path, handlerFn...))
}
func (p party) Delete(path string, handlerFn ...HandlerFunc) IRoute {
	return p.prepared(p._router.Delete(p._rootPath+path, handlerFn...))
}
func (p party) Connect(path string, handlerFn ...HandlerFunc) IRoute {
	return p.prepared(p._router.Connect(p._rootPath+path, handlerFn...))
}
func (p party) Head(path string, handlerFn ...HandlerFunc) IRoute {
	return p._router.Head(p._rootPath+path, handlerFn...)
}
func (p party) Options(path string, handlerFn ...HandlerFunc) IRoute {
	return p.prepared(p._router.Options(p._rootPath+path, handlerFn...))
}
func (p party) Patch(path string, handlerFn ...HandlerFunc) IRoute {
	return p.prepared(p._router.Patch(p._rootPath+path, handlerFn...))
}
func (p party) Trace(path string, handlerFn ...HandlerFunc) IRoute {
	return p._router.Trace(p._rootPath+path, handlerFn...)
}
func (p party) Any(path string, handlerFn ...HandlerFunc) IRoute {
	return p.prepared(p._router.Any(p._rootPath+path, handlerFn...))
}
func (p party) HandleAnnotated(irisHandler Handler) (IRoute, error) {
	route, err := p._router.HandleAnnotated(irisHandler)
	if err != nil {
		return nil, err
	}
	return p.prepared(route), nil
}
func (p party) Handle(method string, registedPath string, handlers ...Handler) IRoute {
	return p.prepared(p._router.Handle(method, registedPath, handlers...))
}
func (p party) HandleFunc(method string, path string, handlerFn ...HandlerFunc) IRoute {
	return p.prepared(p._router.HandleFunc(method, p._rootPath+path, handlerFn...))
}

// Party returns a party of this party, it passes the middleware also
func (p party) Party(path string) IParty {
	joinedParty := NewParty(p._rootPath+path, p._router)
	joinedParty.SetParentHosterMiddleware(p.middleware)
	return joinedParty
}

// Use appends handler(s) to the route or to the router if it's called from router
func (p party) Use(handlers ...Handler) {
	p.middleware = append(p.middleware, handlers...)
}

// UseFunc is the same as Use but it receives HandlerFunc instead of iris.Handler as parameter(s)
// form of acceptable: func(c *iris.Context){//first middleware}, func(c *iris.Context){//second middleware}
func (p party) UseFunc(handlersFn ...HandlerFunc) {
	for _, h := range handlersFn {
		p.Use(Handler(h))
	}
}

var _ IParty = party{}
