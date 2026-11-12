package context

import (
	"io"
	"time"

	"github.com/kataras/iris/v14/core/router/macro"
)

// Router is the interface which all routers should implement.
type Router interface {
	// GetRelPath returns the relative path of the router.
	GetRelPath() string
	// ContextOptions returns the router's request & response options.
	ContextOptions() *ContextOptions

	// Application returns the read-only application.
	// Application() Application
}

// RouteReadOnly allows decoupled access to the current route
// inside the context.
type RouteReadOnly interface {
	// Router returns the Router which owns this route.
	Router() Router

	// GetName returns the route's name.
	GetName() string

	// GetMethod returns the route's method.
	GetMethod() string

	// GetSubdomain returns the route's subdomain.
	GetSubdomain() string

	// GetPath returns the route's original registered path.
	GetPath() string

	// String returns the form of METHOD, SUBDOMAIN, TMPL PATH.
	String() string

	// IsOnline returns true if the route is marked as "online" (state).
	IsOnline() bool

	// IsSitemapDisabled reports whether this route is excluded from the sitemap.
	IsSitemapDisabled() bool

	// IsStatic reports whether this route is a static route.
	// Does not contain dynamic path parameters,
	// is online and registered on GET HTTP Method.
	IsStatic() bool
	// StaticPath returns the static part of the original, registered route path.
	// if /user/{id} it will return /user
	// if /user/{id}/friend/{friendid:uint64} it will return /user too
	// if /assets/{filepath:path} it will return /assets.
	StaticPath() string

	// ResolvePath returns the formatted path's %v replaced with the args.
	ResolvePath(args ...string) string
	// Trace should writes debug route info to the "w".
	// Should be called after Build.
	Trace(w io.Writer, stoppedIndex int)

	// Tmpl returns the path template,
	// it contains the parsed template
	// for the route's path.
	// May contain zero named parameters.
	//
	// Available after the build state, i.e a request handler or Iris Configurator.
	Tmpl() macro.Template

	// GetMainHandlerName returns the name of the route's main handler,
	// the last handler in its MainHandlerList (the one after every middleware).
	GetMainHandlerName() string

	// GetMainHandlerIndex returns the index of the route's main handler
	// inside the MainHandlerList, that is the index of its last handler.
	GetMainHandlerIndex() int

	// GetHandlerList returns the route's handler list.
	// It contains all handlers, middleware and main handlers.
	// It's available only after BuildHandlers method.
	GetHandlerList() *HandlerList
	// GetRegisterFileName returns the file name that this route was registered on.
	GetRegisterFileName() string
	// GetRegisterLineNumber returns the line number that this route was registered on.
	GetRegisterLineNumber() int

	// GetProperty returns a specific property based on its "key"
	// of this route's Router owner.
	GetProperty(key string) (any, bool)

	// IsNoRoute reports whether this route is a no-route (404, fired by the serve mux).
	// Access of the not found path is gain by `ctx.Path()`.
	IsNoRoute() bool
	// IsMethodNotAllowed reports whether this route is a no-method (405, method not allowed, fired by the serve mux).
	// Access of the forbidden request method is gain by `ctx.Method()` and allowed methods through "Allow" response header.
	IsMethodNotAllowed() bool
	// IsPreRouting reports whether this route is running before the main serve mux.
	// Means that this Route's Handlers are running under `UseRouter`.
	IsPreRouting() bool

	// IsRoute reports whether this route is a registered route.
	IsRoute() bool

	// Sitemap properties: https://www.sitemaps.org/protocol.html

	// GetLastMod returns the date of last modification of the file served by this route.
	GetLastMod() time.Time
	// GetChangeFreq returns the the page frequently is likely to change.
	GetChangeFreq() string
	// GetPriority returns the priority of this route's URL relative to other URLs on your site.
	GetPriority() float32
}
