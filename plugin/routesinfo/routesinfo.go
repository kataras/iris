package routesinfo

import (
	"fmt"
	"strings"
	"time"

	"github.com/kataras/iris"
)

//Name the name of the plugin, is "RoutesInfo"
const Name = "RoutesInfo"

// RouteInfo holds the method, domain, path and registered time of a route
type RouteInfo struct {
	Method     string
	Domain     string
	Path       string
	RegistedAt time.Time
}

// String returns the string presentation of the Route(Info)
func (ri RouteInfo) String() string {
	if ri.Domain == "" {
		ri.Domain = "localhost" // only for printing, this doesn't save it, no pointer.
	}
	return fmt.Sprintf("Domain: %s Method: %s Path: %s RegistedAt: %s", ri.Domain, ri.Method, ri.Path, ri.RegistedAt.String())
}

// Plugin the routes info plugin, holds the routes as RouteInfo objects
type Plugin struct {
	routes []RouteInfo
}

// implement the base IPlugin

// GetName ...
func (r Plugin) GetName() string {
	return Name
}

// GetDescription RoutesInfo gives information about the registed routes
func (r Plugin) GetDescription() string {
	return Name + " gives information about the registed routes.\n"
}

//

// implement the rest of the plugin

// PostHandle collect the registed routes information
func (r *Plugin) PostHandle(route iris.IRoute) {
	if r.routes == nil {
		r.routes = make([]RouteInfo, 0)
	}
	r.routes = append(r.routes, RouteInfo{route.GetMethod(), route.GetDomain(), route.GetPath(), time.Now()})
}

// All returns all routeinfos
// returns a slice
func (r Plugin) All() []RouteInfo {
	return r.routes
}

// ByDomain returns all routeinfos which registed to a specific domain
// returns a slice, if nothing founds this slice has 0 len&cap
func (r Plugin) ByDomain(domain string) []RouteInfo {
	var routesByDomain []RouteInfo
	rlen := len(r.routes)
	if domain == "localhost" || domain == "127.0.0.1" || domain == ":" {
		domain = ""
	}
	for i := 0; i < rlen; i++ {
		if r.routes[i].Domain == domain {
			routesByDomain = append(routesByDomain, r.routes[i])
		}
	}
	return routesByDomain
}

// ByMethod returns all routeinfos by a http method
// returns a slice, if nothing founds this slice has 0 len&cap
func (r Plugin) ByMethod(method string) []RouteInfo {
	var routesByMethod []RouteInfo
	rlen := len(r.routes)
	method = strings.ToUpper(method)
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
func (r Plugin) ByPath(path string) []RouteInfo {
	var routesByPath []RouteInfo
	rlen := len(r.routes)
	for i := 0; i < rlen; i++ {
		if r.routes[i].Path == path {
			routesByPath = append(routesByPath, r.routes[i])
		}
	}
	return routesByPath
}

// ByDomainAndMethod returns all routeinfos registed to a specific domain and has specific http method
// returns a slice, if nothing founds this slice has 0 len&cap
func (r Plugin) ByDomainAndMethod(domain string, method string) []RouteInfo {
	var routesByDomainAndMethod []RouteInfo
	rlen := len(r.routes)
	method = strings.ToUpper(method)
	if domain == "localhost" || domain == "127.0.0.1" || domain == ":" {
		domain = ""
	}

	for i := 0; i < rlen; i++ {
		if r.routes[i].Method == method && r.routes[i].Domain == domain {
			routesByDomainAndMethod = append(routesByDomainAndMethod, r.routes[i])
		}
	}
	return routesByDomainAndMethod
}

// ByMethodAndPath returns a single *RouteInfo which has specific http method and path
// returns only the first match
// if nothing founds returns nil
func (r Plugin) ByMethodAndPath(method string, path string) *RouteInfo {

	rlen := len(r.routes)
	for i := 0; i < rlen; i++ {
		if r.routes[i].Method == method && r.routes[i].Path == path {
			return &r.routes[i]
		}
	}
	return nil
}

//
// RoutesInfo returns the Plugin, same as New()
func RoutesInfo() *Plugin {
	return &Plugin{}
}

// New returns the Plugin, same as RoutesInfo()
func New() *Plugin {
	return &Plugin{}
}
