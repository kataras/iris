package context

// RouteReadOnly allows decoupled access to the current route
// inside the context.
type RouteReadOnly interface {
	// Name returns the route's name.
	Name() string
	// String returns the form of METHOD, SUBDOMAIN, TMPL PATH.
	String() string
	// Path returns the route's original registered path.
	Path() string

	// IsOnline returns true if the route is marked as "online" (state).
	IsOnline() bool

	// ResolvePath returns the formatted path's %v replaced with the args.
	ResolvePath(args ...string) string
}
