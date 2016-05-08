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
	"strings"
)

type (
	// IRoute is the interface which the Route should implements
	// it useful to have it as an interface because this interface is passed to the plugins
	IRoute interface {
		GetMethod() string
		GetDomain() string
		GetPath() string
		GetPathPrefix() string
		ProcessPath()
		GetMiddleware() Middleware
		SetMiddleware(m Middleware)
		HasCors() bool
	}

	// Route contains basic and temporary info about the route, it is nil after iris.Listen called
	// contains all middleware and prepare them for execution
	// Used to create a node at the Router's Build state
	Route struct {
		method     string
		domain     string
		fullpath   string
		PathPrefix string
		middleware Middleware
	}
)

var _ IRoute = &Route{}

// NewRoute creates, from a path string, and a slice of HandlerFunc
func NewRoute(method string, registedPath string, middleware Middleware) *Route {
	domain := ""
	if registedPath[0] != SlashByte && strings.Contains(registedPath, ".") && (strings.IndexByte(registedPath, SlashByte) == -1 || strings.IndexByte(registedPath, SlashByte) > strings.IndexByte(registedPath, '.')) {
		//means that is a path with domain
		//we have to extract the domain

		//find the first '/'
		firstSlashIndex := strings.IndexByte(registedPath, SlashByte)

		//firt of all remove the first '/' if that exists and we have domain
		if firstSlashIndex == 0 {
			//e.g /admin.ideopod.com/hey
			//then just remove the first slash and re-execute the NewRoute and return it
			registedPath = registedPath[1:]
			return NewRoute(method, registedPath, middleware)
		}
		//if it's just the domain, then set it(registedPath) as the domain
		//and after set the registedPath to a slash '/' for the path part
		if firstSlashIndex == -1 {
			domain = registedPath
			registedPath = Slash
		} else {
			//we have a domain + path
			domain = registedPath[0:firstSlashIndex]
			registedPath = registedPath[len(domain):]
		}

	}
	r := &Route{method: method, domain: domain, fullpath: registedPath, middleware: middleware}
	r.ProcessPath()
	return r
}

// GetMethod returns the http method
func (r Route) GetMethod() string {
	return r.method
}

// GetDomain returns the registed domain which this route is ( if none, is "" which is means "localhost"/127.0.0.1)
func (r Route) GetDomain() string {
	return r.domain
}

// GetPath returns the full registed path
func (r Route) GetPath() string {
	return r.fullpath
}

// GetPathPrefix returns the path prefix, this is the static part before any parameter or *any
func (r Route) GetPathPrefix() string {
	return r.PathPrefix
}

// GetMiddleware returns the chain of the []HandlerFunc registed to this Route
func (r Route) GetMiddleware() Middleware {
	return r.middleware
}

// SetMiddleware sets the middleware(s)
func (r Route) SetMiddleware(m Middleware) {
	r.middleware = m
}

// ProcessPath modifies the path in order to set the path prefix of this Route
func (r *Route) ProcessPath() {
	endPrefixIndex := strings.IndexByte(r.fullpath, ParameterStartByte)

	if endPrefixIndex != -1 {
		r.PathPrefix = r.fullpath[:endPrefixIndex]

	} else {
		//check for *
		endPrefixIndex = strings.IndexByte(r.fullpath, MatchEverythingByte)
		if endPrefixIndex != -1 {
			r.PathPrefix = r.fullpath[:endPrefixIndex]
		} else {
			//check for the last slash
			endPrefixIndex = strings.LastIndexByte(r.fullpath, SlashByte)
			if endPrefixIndex != -1 {
				r.PathPrefix = r.fullpath[:endPrefixIndex]
			} else {
				//we don't have ending slash ? then it is the whole r.fullpath
				r.PathPrefix = r.fullpath
			}
		}
	}

	//1.check if pathprefix is empty ( it's empty when we have just '/' as fullpath) so make it '/'
	//2. check if it's not ending with '/', ( it is not ending with '/' when the next part is parameter or *)

	lastIndexOfSlash := strings.LastIndexByte(r.PathPrefix, SlashByte)
	if lastIndexOfSlash != len(r.PathPrefix)-1 || r.PathPrefix == "" {
		r.PathPrefix += "/"
	}
}

// HasCors check if middleware passsed to a route has cors
func (r *Route) HasCors() bool {
	return RouteConflicts(r, "httpmethod")
}

// RouteConflicts checks for route's middleware conflicts
func RouteConflicts(r *Route, with string) bool {
	for _, h := range r.middleware {
		if m, ok := h.(interface {
			Conflicts() string
		}); ok {
			if c := m.Conflicts(); c == with {
				return true
			}
		}
	}
	return false
}
