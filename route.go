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

type IRoute interface {
	ProcessPath()
	GetDomain() string
	GetPath() string
	GetPathPrefix() string
	GetMiddleware() Middleware
	SetMiddleware(m Middleware)
}

// Route contains basic and temporary info about the route, it is nil after iris.Listen called
// contains all middleware and prepare them for execution
// Used to create a node at the Router's Build state
type Route struct {
	middleware Middleware
	domain     string
	fullpath   string
	PathPrefix string
}

var _ IRoute = &Route{}

// newRoute creates, from a path string, and a slice of HandlerFunc
func NewRoute(registedPath string, middleware Middleware) *Route {
	domain := ""
	if strings.Contains(registedPath, ".") {
		//means that is a path with domain
		//we have to extract the domain

		//find the first '/'
		firstSlashIndex := strings.IndexByte(registedPath, SlashByte)

		//firt of all remove the first '/' if that exists and we have domain
		if firstSlashIndex == 0 {
			//e.g /admin.ideopod.com/hey
			//then just remove the first slash and re-execute the NewRoute and return it
			registedPath = registedPath[1:]
			return NewRoute(registedPath, middleware)
		}
		//if it's just the domain, then set it(registedPath) as the domain
		//and after set the registedPath to a slash '/' for the path part
		if firstSlashIndex == -1 {
			domain = registedPath
			registedPath = "/"
		} else {
			//we have a domain + path
			domain = registedPath[0:firstSlashIndex]
			registedPath = registedPath[len(domain):]
		}

	}
	r := &Route{fullpath: registedPath, domain: domain}
	r.middleware = middleware
	r.ProcessPath()
	return r
}
func (r Route) GetDomain() string {
	return r.domain
}
func (r Route) GetPath() string {
	return r.fullpath
}

func (r Route) GetPathPrefix() string {
	return r.PathPrefix
}

func (r Route) GetMiddleware() Middleware {
	return r.middleware
}

func (r Route) SetMiddleware(m Middleware) {
	r.middleware = m
}

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
