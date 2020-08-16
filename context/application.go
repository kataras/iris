package context

import (
	stdContext "context"
	"io"
	"net/http"
	"sync"

	"github.com/kataras/golog"
	"github.com/tdewolff/minify/v2"
)

// Application is the context's owner.
// This interface contains the functions that can be used with safety inside a Handler
// by `context.Application()`.
type Application interface {
	// ConfigurationReadOnly returns all the available configuration values can be used on a request.
	ConfigurationReadOnly() ConfigurationReadOnly

	// Logger returns the golog logger instance(pointer) that is being used inside the "app".
	Logger() *golog.Logger

	// I18nReadOnly returns the i18n's read-only features.
	I18nReadOnly() I18nReadOnly

	// Validate validates a value and returns nil if passed or
	// the failure reason if not.
	Validate(interface{}) error

	// Minifier returns the minifier instance.
	// By default it can minifies:
	// - text/html
	// - text/css
	// - image/svg+xml
	// - application/text(javascript, ecmascript, json, xml).
	// Use that instance to add custom Minifiers before server ran.
	Minifier() *minify.M
	// View executes and write the result of a template file to the writer.
	//
	// Use context.View to render templates to the client instead.
	// Returns an error on failure, otherwise nil.
	View(writer io.Writer, filename string, layout string, bindingData interface{}) error

	// ServeHTTPC is the internal router, it's visible because it can be used for advanced use cases,
	// i.e: routing within a foreign context.
	//
	// It is ready to use after Build state.
	ServeHTTPC(ctx *Context)

	// ServeHTTP is the main router handler which calls the .Serve and acquires a new context from the pool.
	//
	// It is ready to use after Build state.
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// Shutdown gracefully terminates all the application's server hosts and any tunnels.
	// Returns an error on the first failure, otherwise nil.
	Shutdown(ctx stdContext.Context) error

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

	// FireErrorCode handles the response's error response.
	// If `Configuration.ResetOnFireErrorCode()` is true
	// and the response writer was a recorder or a gzip writer one
	// then it will try to reset the headers and the body before calling the
	// registered (or default) error handler for that error code set by
	// `ctx.StatusCode` method.
	FireErrorCode(ctx *Context)

	// RouteExists reports whether a particular route exists
	// It will search from the current subdomain of context's host, if not inside the root domain.
	RouteExists(ctx *Context, method, path string) bool
	// FindClosestPaths returns a list of "n" paths close to "path" under the given "subdomain".
	//
	// Order may change.
	FindClosestPaths(subdomain, searchPath string, n int) []string
}

var (
	registeredApps []Application
	mu             sync.RWMutex
)

// RegisterApplication registers an application to the global shared storage.
func RegisterApplication(app Application) {
	if app == nil {
		return
	}

	mu.Lock()
	registeredApps = append(registeredApps, app)
	mu.Unlock()
}

// LastApplication returns the last registered Application.
// Handlers has access to the current Application,
// use `Context.Application()` instead.
func LastApplication() Application {
	mu.RLock()
	if n := len(registeredApps); n > 0 {
		if app := registeredApps[n-1]; app != nil {
			mu.RUnlock()
			return app
		}
	}
	mu.RUnlock()
	return nil
}

// DefaultLogger returns a Logger instance for an Iris module.
// If the program contains at least one registered Iris Application
// before this call then it will return a child of that Application's Logger
// otherwise a fresh child of the `golog.Default` will be returned instead.
//
// It should be used when a module has no access to the Application or its Logger.
func DefaultLogger(prefix string) (logger *golog.Logger) {
	if app := LastApplication(); app != nil {
		logger = app.Logger()
	} else {
		logger = golog.Default
	}

	logger = logger.Child(prefix)
	return
}
