package context

import (
	"io"
	"time"

	"github.com/kataras/iris/v12/macro"
)

// RouteReadOnly allows decoupled access to the current route
// inside the context.
type RouteReadOnly interface {
	// Name returns the route's name.
	Name() string

	// StatusErrorCode returns 0 for common resource routes
	// or the error code that an http error handler registered on.
	StatusErrorCode() int

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

	// MainHandlerName returns the first registered handler for the route.
	MainHandlerName() string

	// MainHandlerIndex returns the first registered handler's index for the route.
	MainHandlerIndex() int

	// Property returns a specific property based on its "key"
	// of this route's Party owner.
	Property(key string) (interface{}, bool)

	// Sitemap properties: https://www.sitemaps.org/protocol.html

	// GetLastMod returns the date of last modification of the file served by this route.
	GetLastMod() time.Time
	// GetChangeFreq returns the the page frequently is likely to change.
	GetChangeFreq() string
	// GetPriority returns the priority of this route's URL relative to other URLs on your site.
	GetPriority() float32
}
