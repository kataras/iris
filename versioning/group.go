package versioning

import (
	"net/http"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
)

type (
	vroute struct {
		method   string
		path     string
		versions Map
	}

	// Group is a group of version-based routes.
	// One version per one or more routes.
	Group struct {
		version      string
		extraMethods []string
		routes       []vroute

		deprecation DeprecationOptions
	}
)

// NewGroup returns a ptr to Group based on the given "version".
//
// See `Handle` and `RegisterGroups` for more.
func NewGroup(version string) *Group {
	return &Group{
		version: version,
	}
}

// Deprecated marks this group and all its versioned routes
// as deprecated versions of that endpoint.
// It can be called in the end just before `RegisterGroups`
// or first by `NewGroup(...).Deprecated(...)`. It returns itself.
func (g *Group) Deprecated(options DeprecationOptions) *Group {
	// if `Deprecated` is called in the end.
	for _, r := range g.routes {
		r.versions[g.version] = Deprecated(r.versions[g.version], options)
	}

	// store the options if called before registering any versioned routes.
	g.deprecation = options

	return g
}

// AllowMethods can be called before `Handle/Get/Post...`
// to tell the underline router that all routes should be registered
// to these "methods" as well.
func (g *Group) AllowMethods(methods ...string) *Group {
	g.extraMethods = append(g.extraMethods, methods...)
	return g
}

func (g *Group) addVRoute(method, path string, handler context.Handler) {
	for _, r := range g.routes { // check if route already exists.
		if r.method == method && r.path == path {
			return
		}
	}

	g.routes = append(g.routes, vroute{
		method:   method,
		path:     path,
		versions: Map{g.version: handler},
	})
}

// Handle registers a versioned route to the group.
// A call of `RegisterGroups` is necessary in order to register the actual routes
// when the group is complete.
//
// `RegisterGroups` for more.
func (g *Group) Handle(method string, path string, handler context.Handler) {
	if g.deprecation.ShouldHandle() { // if `Deprecated` called first.
		handler = Deprecated(handler, g.deprecation)
	}

	methods := append(g.extraMethods, method)

	for _, method := range methods {
		g.addVRoute(method, path, handler)
	}
}

// None registers an "offline" versioned route
// see `context#ExecRoute(routeName)` and routing examples.
func (g *Group) None(path string, handler context.Handler) {
	g.Handle(router.MethodNone, path, handler)
}

// Get registers a versioned route for the Get http method.
func (g *Group) Get(path string, handler context.Handler) {
	g.Handle(http.MethodGet, path, handler)
}

// Post registers a versioned route for the Post http method.
func (g *Group) Post(path string, handler context.Handler) {
	g.Handle(http.MethodPost, path, handler)
}

// Put registers a versioned route for the Put http method
func (g *Group) Put(path string, handler context.Handler) {
	g.Handle(http.MethodPut, path, handler)
}

// Delete registers a versioned route for the Delete http method.
func (g *Group) Delete(path string, handler context.Handler) {
	g.Handle(http.MethodDelete, path, handler)
}

// Connect registers a versioned route for the Connect http method.
func (g *Group) Connect(path string, handler context.Handler) {
	g.Handle(http.MethodConnect, path, handler)
}

// Head registers a versioned route for the Head http method.
func (g *Group) Head(path string, handler context.Handler) {
	g.Handle(http.MethodHead, path, handler)
}

// Options registers a versioned route for the Options http method.
func (g *Group) Options(path string, handler context.Handler) {
	g.Handle(http.MethodOptions, path, handler)
}

// Patch registers a versioned route for the Patch http method.
func (g *Group) Patch(path string, handler context.Handler) {
	g.Handle(http.MethodPatch, path, handler)
}

// Trace registers a versioned route for the Trace http method.
func (g *Group) Trace(path string, handler context.Handler) {
	g.Handle(http.MethodTrace, path, handler)
}

// Any registers a versioned route for ALL of the http methods
// (Get,Post,Put,Head,Patch,Options,Connect,Delete).
func (g *Group) Any(registeredPath string, handler context.Handler) {
	g.Get(registeredPath, handler)
	g.Post(registeredPath, handler)
	g.Put(registeredPath, handler)
	g.Delete(registeredPath, handler)
	g.Connect(registeredPath, handler)
	g.Head(registeredPath, handler)
	g.Options(registeredPath, handler)
	g.Patch(registeredPath, handler)
	g.Trace(registeredPath, handler)
}

// RegisterGroups registers one or more groups to an `iris.Party` or to the root router.
// See `NewGroup` and `NotFoundHandler` too.
func RegisterGroups(r router.Party, notFoundHandler context.Handler, groups ...*Group) (actualRoutes []*router.Route) {
	var total []vroute
	for _, g := range groups {
	inner:
		for _, r := range g.routes {
			for i, tr := range total {
				if tr.method == r.method && tr.path == r.path {
					total[i].versions[g.version] = r.versions[g.version]
					continue inner
				}
			}

			total = append(total, r)
		}
	}

	for _, vr := range total {
		if notFoundHandler != nil {
			vr.versions[NotFound] = notFoundHandler
		}

		route := r.Handle(vr.method, vr.path, NewMatcher(vr.versions))
		actualRoutes = append(actualRoutes, route)
	}

	return
}
