package context

import (
	"io"
	"net/http"

	"github.com/kataras/golog"
)

// Application is the context's owner.
// This interface contains the functions that can be used with safety inside a Handler
// by `context.Application()`.
type Application interface {
	// ConfigurationReadOnly returns all the available configuration values can be used on a request.
	ConfigurationReadOnly() ConfigurationReadOnly

	// Logger returns the golog logger instance(pointer) that is being used inside the "app".
	Logger() *golog.Logger

	// View executes and write the result of a template file to the writer.
	//
	// Use context.View to render templates to the client instead.
	// Returns an error on failure, otherwise nil.
	View(writer io.Writer, filename string, layout string, bindingData interface{}) error

	// ServeHTTPC is the internal router, it's visible because it can be used for advanced use cases,
	// i.e: routing within a foreign context.
	//
	// It is ready to use after Build state.
	ServeHTTPC(ctx Context)

	// ServeHTTP is the main router handler which calls the .Serve and acquires a new context from the pool.
	//
	// It is ready to use after Build state.
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// GetRouteReadOnly returns the registered "read-only" route based on its name, otherwise nil.
	// One note: "routeName" should be case-sensitive. Used by the context to get the current route.
	// It returns an interface instead to reduce wrong usage and to keep the decoupled design between
	// the context and the routes.
	//
	// Look core/router/APIBuilder#GetRoute for more.
	GetRouteReadOnly(routeName string) RouteReadOnly

	// GetRoutesReadOnly returns the registered "read-only" routes.
	//
	// Look core/router/APIBuilder#GetRoutes for more.
	GetRoutesReadOnly() []RouteReadOnly

	// FireErrorCode executes an error http status code handler
	// based on the context's status code.
	//
	// If a handler is not already registered,
	// then it creates & registers a new trivial handler on the-fly.
	FireErrorCode(ctx Context)

	// RouteExists reports whether a particular route exists
	// It will search from the current subdomain of context's host, if not inside the root domain.
	RouteExists(ctx Context, method, path string) bool
}
