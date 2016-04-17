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
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package routesinfo

import (
	"fmt"
	"strings"
	"time"

	"github.com/kataras/iris"
)

//the name of the plugin
var Name = "RoutesInfo"

type RouteInfo struct {
	Method     string
	Domain     string
	Path       string
	RegistedAt time.Time
}

func (ri RouteInfo) String() string {
	if ri.Domain == "" {
		ri.Domain = "localhost" // only for printing, this doesn't save it, no pointer.
	}
	return fmt.Sprintf("Domain: %s Method: %s Path: %s RegistedAt: %s", ri.Domain, ri.Method, ri.Path, ri.RegistedAt.String())
}

type RoutesInfoPlugin struct {
	routes []RouteInfo
}

// implement the base IPlugin
func (r *RoutesInfoPlugin) Activate(container iris.IPluginContainer) error {
	return nil
}

func (r RoutesInfoPlugin) GetName() string {
	return Name
}

func (r RoutesInfoPlugin) GetDescription() string {
	return Name + " gives information about the registed routes.\n"
}

//

// implement the rest of the plugin

// PostHandle collect the registed routes information
func (r *RoutesInfoPlugin) PostHandle(route iris.IRoute) {
	if r.routes == nil {
		r.routes = make([]RouteInfo, 0)
	}
	r.routes = append(r.routes, RouteInfo{route.GetMethod(), route.GetDomain(), route.GetPath(), time.Now()})
}

// All returns all routeinfos
// returns a slice
func (r RoutesInfoPlugin) All() []RouteInfo {
	return r.routes
}

// ByDomain returns all routeinfos which registed to a specific domain
// returns a slice, if nothing founds this slice has 0 len&cap
func (r RoutesInfoPlugin) ByDomain(domain string) []RouteInfo {
	rlen := len(r.routes)
	if domain == "localhost" || domain == "127.0.0.1" || domain == ":" {
		domain = ""
	}
	routesByDomain := make([]RouteInfo, 0)
	for i := 0; i < rlen; i++ {
		if r.routes[i].Domain == domain {
			routesByDomain = append(routesByDomain, r.routes[i])
		}
	}
	return routesByDomain
}

// ByMethod returns all routeinfos by a http method
// returns a slice, if nothing founds this slice has 0 len&cap
func (r RoutesInfoPlugin) ByMethod(method string) []RouteInfo {
	rlen := len(r.routes)
	method = strings.ToUpper(method)
	routesByMethod := make([]RouteInfo, 0)
	for i := 0; i < rlen; i++ {
		if r.routes[i].Method == method {
			routesByMethod = append(routesByMethod, r.routes[i])
		}
	}
	return routesByMethod
}

// ByPath returns all routeinfos by a path
// maybe one path is the same on GET and POST ( for example /login GET, /login POST)
// because of that it returns a slice and not only one RouteInfo
// returns a slice, if nothing founds this slice has 0 len&cap
func (r RoutesInfoPlugin) ByPath(path string) []RouteInfo {
	rlen := len(r.routes)
	routesByPath := make([]RouteInfo, 0)
	for i := 0; i < rlen; i++ {
		if r.routes[i].Path == path {
			routesByPath = append(routesByPath, r.routes[i])
		}
	}
	return routesByPath
}

// ByDomainAndMethod returns all routeinfos registed to a specific domain and has specific http method
// returns a slice, if nothing founds this slice has 0 len&cap
func (r RoutesInfoPlugin) ByDomainAndMethod(domain string, method string) []RouteInfo {
	rlen := len(r.routes)
	method = strings.ToUpper(method)
	if domain == "localhost" || domain == "127.0.0.1" || domain == ":" {
		domain = ""
	}
	routesByDomainAndMethod := make([]RouteInfo, 0)
	for i := 0; i < rlen; i++ {
		if r.routes[i].Method == method && r.routes[i].Domain == domain {
			routesByDomainAndMethod = append(routesByDomainAndMethod, r.routes[i])
		}
	}
	return routesByDomainAndMethod
}

// ByPathAndMehod returns a single *RouteInfo which has specific http method and path
// returns only the first match
// if nothing founds returns nil
func (r RoutesInfoPlugin) ByMethodAndPath(method string, path string) *RouteInfo {
	rlen := len(r.routes)
	for i := 0; i < rlen; i++ {
		if r.routes[i].Method == method && r.routes[i].Path == path {
			return &r.routes[i]
		}
	}
	return nil
}

//
// RoutesInfo returns the RoutesInfoPlugin
func RoutesInfo() *RoutesInfoPlugin {
	return &RoutesInfoPlugin{}
}
