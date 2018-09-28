package context

import "github.com/kataras/iris/macro"

// RouteReadOnly allows decoupled access to the current route
// inside the context.
type RouteReadOnly interface {
	// Name returns the route's name.
	Name() string

	// Method returns the route's method.
	Method() string

	// Subdomains returns the route's subdomain.
	Subdomain() string

	// Path returns the route's original registered path.
	Path() string

	// String returns the form of METHOD, SUBDOMAIN, TMPL PATH.
	String() string

	// IsOnline returns true if the route is marked as "online" (state).
	IsOnline() bool

	// StaticPath returns the static part of the original, registered route path.
	// if /user/{id} it will return /user
	// if /user/{id}/friend/{friendid:uint64} it will return /user too
	// if /assets/{filepath:path} it will return /assets.
	StaticPath() string

	// ResolvePath returns the formatted path's %v replaced with the args.
	ResolvePath(args ...string) string

	// Tmpl returns the path template,
	// it contains the parsed template
	// for the route's path.
	// May contain zero named parameters.
	//
	// Available after the build state, i.e a request handler or Iris Configurator.
	Tmpl() macro.Template

	// MainHandlerName returns the first registered handler for the route.
	MainHandlerName() string
}
