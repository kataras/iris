package iris

import (
	"sort"
	"strings"
)

type (
	// RouteInfo is just the (other idea was : RouteInfo but we needed the Change/SetName visible so...)
	// information of the registered routes.
	RouteInfo interface {
		// ChangeName & AllowOPTIONS are the only one route property
		// which can be change without any bad side-affects
		// so it's the only setter here.
		//
		// It's used on iris.Default.Handle()
		ChangeName(name string) RouteInfo

		// Name returns the name of the route
		Name() string
		// Method returns the http method
		Method() string
		// AllowOPTIONS called when this route is targeting OPTIONS methods too
		// it's an alternative way of registring the same route with '.OPTIONS("/routepath", routeMiddleware)'
		AllowOPTIONS() RouteInfo
		// HasCors returns true if the route is targeting OPTIONS methods too
		// or it has a middleware which conflicts with "httpmethod",
		// otherwise false
		HasCors() bool
		// Subdomain returns the subdomain,if any
		Subdomain() string
		// Path returns the path
		Path() string
		// Middleware returns the slice of Handler([]Handler) registered to this route
		Middleware() Middleware
		// IsOnline returns true if the route is marked as "online" (state)
		IsOnline() bool
	}

	// route holds  useful information about route
	route struct {
		// if no name given then it's the subdomain+path
		name               string
		subdomain          string
		method             string
		allowOptionsMethod bool
		path               string
		middleware         Middleware
	}
)

var _ RouteInfo = &route{}

// RouteConflicts checks for route's middleware conflicts
func RouteConflicts(r RouteInfo, with string) bool {
	for _, h := range r.Middleware() {
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

// Name returns the name of the route
func (r route) Name() string {
	return r.name
}

// Name returns the name of the route
func (r *route) ChangeName(name string) RouteInfo {
	r.name = name
	return r
}

// AllowOPTIONS called when this route is targeting OPTIONS methods too
// it's an alternative way of registring the same route with '.OPTIONS("/routepath", routeMiddleware)'
func (r *route) AllowOPTIONS() RouteInfo {
	r.allowOptionsMethod = true
	return r
}

// Method returns the http method
func (r route) Method() string {
	return r.method
}

// Subdomain returns the subdomain,if any
func (r route) Subdomain() string {
	return r.subdomain
}

// Path returns the path
func (r route) Path() string {
	return r.path
}

// Middleware returns the slice of Handler([]Handler) registered to this route
func (r route) Middleware() Middleware {
	return r.middleware
}

// IsOnline returns true if the route is marked as "online" (state)
func (r route) IsOnline() bool {
	return r.method != MethodNone
}

// HasCors returns true if the route is targeting OPTIONS methods too
// or it has a middleware which conflicts with "httpmethod",
// otherwise false
func (r *route) HasCors() bool {
	return r.allowOptionsMethod || RouteConflicts(r, "httpmethod")
}

// MethodChangedListener listener signature fired when route method changes
type MethodChangedListener func(routeInfo RouteInfo, oldMethod string)

// RouteRepository contains the interface which is used on custom routers
// contains methods and helpers to find a route by its name,
// and change its method, path, middleware.
//
// This is not visible outside except the RouterBuilderPolicy
type RouteRepository interface { // RouteEngine  kai ContextEngine mesa sto builder adi gia RouteRepository kai ContextEngine
	RoutesInfo
	ChangeName(routeInfo RouteInfo, newName string)
	ChangeMethod(routeInfo RouteInfo, newMethod string)
	ChangePath(routeInfo RouteInfo, newPath string)
	ChangeMiddleware(routeInfo RouteInfo, newMiddleware Middleware)
}

// RoutesInfo is the interface which contains the valid actions
// permitted at RUNTIME
type RoutesInfo interface { // RouteRepository
	Lookup(routeName string) RouteInfo
	Visit(visitor func(RouteInfo))
	OnMethodChanged(methodChangedListener MethodChangedListener)
	Online(routeInfo RouteInfo, HTTPMethod string) bool
	Offline(routeInfo RouteInfo) bool
}

// routeRepository contains all the routes.
// Implements both RouteRepository and RoutesInfo
type routeRepository struct {
	routes []*route
	// when builded (TODO: move to its own struct)
	methodChangedListeners []MethodChangedListener
}

var _ sort.Interface = &routeRepository{}
var _ RouteRepository = &routeRepository{}

// Len is the number of elements in the collection.
func (r routeRepository) Len() int {
	return len(r.routes)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (r routeRepository) Less(i, j int) bool {
	return len(r.routes[i].Subdomain()) > len(r.routes[j].Subdomain())
}

// Swap swaps the elements with indexes i and j.
func (r routeRepository) Swap(i, j int) {
	r.routes[i], r.routes[j] = r.routes[j], r.routes[i]
}

func (r *routeRepository) register(method, subdomain, path string,
	middleware Middleware) *route {

	_route := &route{
		name:       method + subdomain + path,
		method:     method,
		subdomain:  subdomain,
		path:       path,
		middleware: middleware,
	}

	r.routes = append(r.routes, _route)
	return _route
}

func (r *routeRepository) getRouteByName(routeName string) *route {
	for i := range r.routes {
		_route := r.routes[i]
		if _route.name == routeName {
			return _route
		}
	}
	return nil
}

// Lookup returns a route by its name
// used for reverse routing and templates
func (r *routeRepository) Lookup(routeName string) RouteInfo {
	route := r.getRouteByName(routeName)
	if route == nil {
		return nil
	}
	return route
}

// ChangeName changes the Name of an existing route
func (r *routeRepository) ChangeName(routeInfo RouteInfo,
	newName string) {

	if newName != "" {
		route := r.getRouteByName(routeInfo.Name())
		if route != nil {
			route.name = newName
		}
	}
}

func (r *routeRepository) OnMethodChanged(methodChangedListener MethodChangedListener) {
	r.methodChangedListeners = append(r.methodChangedListeners, methodChangedListener)
}

func (r *routeRepository) fireMethodChangedListeners(routeInfo RouteInfo, oldMethod string) {
	for i := 0; i < len(r.methodChangedListeners); i++ {
		r.methodChangedListeners[i](routeInfo, oldMethod)
	}
}

// ChangeMethod changes the Method of an existing route
func (r *routeRepository) ChangeMethod(routeInfo RouteInfo,
	newMethod string) {
	newMethod = strings.ToUpper(newMethod)
	valid := false
	for _, m := range AllMethods {
		if newMethod == m || newMethod == MethodNone {
			valid = true
		}
	}

	if valid {

		route := r.getRouteByName(routeInfo.Name())
		if route != nil && route.method != newMethod {
			oldMethod := route.method
			route.method = newMethod
			r.fireMethodChangedListeners(routeInfo, oldMethod)
		}
	}
}

// Online sets the state of the route to "online" with a specific http method
// it re-builds the router
//
// returns true if state was actually changed
//
// see context.ExecRoute(routeInfo),
// iris.Default.None(...) and iris.Routes.Online/.Routes.Offline
// For more details look: https://github.com/kataras/iris/issues/585
//
// Example: https://github.com/iris-contrib/examples/tree/master/route_state
func (r *routeRepository) Online(routeInfo RouteInfo, HTTPMethod string) bool {
	return r.changeRouteState(routeInfo, HTTPMethod)
}

// Offline sets the state of the route to "offline" and re-builds the router
//
// returns true if state was actually changed
//
// see context.ExecRoute(routeInfo),
// iris.Default.None(...) and iris.Routes.Online/.Routes.Offline
// For more details look: https://github.com/kataras/iris/issues/585
//
// Example: https://github.com/iris-contrib/examples/tree/master/route_state
func (r *routeRepository) Offline(routeInfo RouteInfo) bool {
	return r.changeRouteState(routeInfo, MethodNone)
}

// changeRouteState changes the state of the route.
// iris.MethodNone for offline
// and iris.MethodGet/MethodPost/MethodPut/MethodDelete /MethodConnect/MethodOptions/MethodHead/MethodTrace/MethodPatch for online
// it re-builds the router
//
// returns true if state was actually changed
func (r *routeRepository) changeRouteState(routeInfo RouteInfo, HTTPMethod string) bool {
	if routeInfo != nil {
		nonSpecificMethod := len(HTTPMethod) == 0
		if routeInfo.Method() != HTTPMethod {
			if nonSpecificMethod {
				r.ChangeMethod(routeInfo, MethodGet) // if no method given, then do it for "GET" only
			} else {
				r.ChangeMethod(routeInfo, HTTPMethod)
			}
			// re-build the router/main handler should be implemented
			// on the custom router via OnMethodChanged event.
			return true
		}
	}
	return false
}

// ChangePath changes the Path of an existing route
func (r *routeRepository) ChangePath(routeInfo RouteInfo,
	newPath string) {

	if newPath != "" {
		route := r.getRouteByName(routeInfo.Name())
		if route != nil {
			route.path = newPath
		}
	}
}

// ChangeMiddleware changes the Middleware/Handlers of an existing route
func (r *routeRepository) ChangeMiddleware(routeInfo RouteInfo,
	newMiddleware Middleware) {

	route := r.getRouteByName(routeInfo.Name())
	if route != nil {
		route.middleware = newMiddleware
	}
}

// Visit accepts a visitor func which receives a route(readonly).
// That visitor func accepts the next route of each of the route entries.
func (r *routeRepository) Visit(visitor func(RouteInfo)) {
	for i := range r.routes {
		visitor(r.routes[i])
	}
}

// sort sorts routes by subdomain.
func (r *routeRepository) sort() {
	sort.Sort(r)
}
