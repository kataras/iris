package routesinfo

import (
	"github.com/kataras/iris"
	"strings"
	"time"
)

//the name of the plugin
var Name = "RoutesInfo"

type RouteInfo struct {
	Method     string
	Domain     string
	Path       string
	RegistedAt time.Time
}

type routesinfoPlugin struct {
	routes []RouteInfo
}

// implement the base IPlugin
func (r *routesinfoPlugin) Activate(container iris.IPluginContainer) error {
	return nil
}

func (r routesinfoPlugin) GetName() string {
	return Name
}

func (r routesinfoPlugin) GetDescription() string {
	return Name + " gives information about the registed routes.\n"
}

//

// implement the rest of the plugin

// PostHandle collect the registed routes information
func (r *routesinfoPlugin) PostHandle(route iris.IRoute) {
	if r.routes == nil {
		r.routes = make([]RouteInfo, 0)
	}
	r.routes = append(r.routes, RouteInfo{route.GetMethod(), route.GetDomain(), route.GetPath(), time.Now()})
}

// All returns all routeinfos
// returns a slice
func (r routesinfoPlugin) All() []RouteInfo {
	return r.routes
}

// ByDomain returns all routeinfos which registed to a specific domain
// returns a slice, if nothing founds this slice has 0 len&cap
func (r routesinfoPlugin) ByDomain(domain string) []RouteInfo {
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
func (r routesinfoPlugin) ByMethod(method string) []RouteInfo {
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
func (r routesinfoPlugin) ByPath(path string) []RouteInfo {
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
func (r routesinfoPlugin) ByDomainAndMethod(domain string, method string) []RouteInfo {
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
func (r routesinfoPlugin) ByMethodAndPath(method string, path string) *RouteInfo {
	rlen := len(r.routes)
	for i := 0; i < rlen; i++ {
		if r.routes[i].Method == method && r.routes[i].Path == path {
			return &r.routes[i]
		}
	}
	return nil
}

//
// RoutesInfo returns the routesInfoPlugin
func RoutesInfo() *routesinfoPlugin {
	return &routesinfoPlugin{}
}
