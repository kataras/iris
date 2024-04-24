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
	// IsDebug reports whether the application is running
	// under debug/development mode.
	// It's just a shortcut of Logger().Level >= golog.DebugLevel.
	// The same method existss as Context.IsDebug() too.
	IsDebug() bool

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

	// GetContextPool returns the Iris sync.Pool which holds the contexts values.
	// Iris automatically releases the request context, so you don't have to use it.
	// It's only useful to manually release the context on cases that connection
	// is hijacked by a third-party middleware and the http handler return too fast.
	GetContextPool() *Pool

	// GetContextErrorHandler returns the handler which handles errors
	// on JSON write failures.
	GetContextErrorHandler() ErrorHandler

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

	// String returns the Application's Name.
	String() string
}

// Notes(@kataras):
// Alternative places...
// 1. in apps/store, but it would require an empty `import _ "....apps/store"
//	  from end-developers, to avoid the import cycle and *iris.Application access.
// 2. in root package level, that could be the best option, it has access to the *iris.Application
// instead of the context.Application interface, but we try to keep the root package
// as minimum as possible, however: if in the future, those Application instances
// can be registered through network instead of same-process then we must think of that choice.
// 3. this is the best possible place, as the root package and all subpackages
// have access to this context package without import cycles and they already using it,
// the only downside is that we don't have access to the *iris.Application instance
// but this context.Application is designed that way that can execute all important methods
// as the whole Iris code base is so well written.

var (
	// registerApps holds all the created iris Applications by this process.
	// It's slice instead of map because if IRIS_APP_NAME env var exists,
	// by-default all applications running on the same machine
	// will have the same name unless `Application.SetName` is called.
	registeredApps                   []Application
	onApplicationRegisteredListeners []func(Application)
	mu                               sync.RWMutex
)

// RegisterApplication registers an application to the global shared storage.
func RegisterApplication(app Application) {
	if app == nil {
		return
	}

	mu.Lock()
	registeredApps = append(registeredApps, app)
	mu.Unlock()

	mu.RLock()
	for _, listener := range onApplicationRegisteredListeners {
		listener(app)
	}
	mu.RUnlock()
}

// OnApplicationRegistered adds a function which fires when a new application
// is registered.
func OnApplicationRegistered(listeners ...func(app Application)) {
	mu.Lock()
	onApplicationRegisteredListeners = append(onApplicationRegisteredListeners, listeners...)
	mu.Unlock()
}

// GetApplications returns a slice of all the registered Applications.
func GetApplications() []Application {
	mu.RLock()
	// a copy slice but the instances are pointers so be careful what modifications are done
	// the return value is read-only but it can be casted to *iris.Application.
	apps := make([]Application, 0, len(registeredApps))
	copy(apps, registeredApps)
	mu.RUnlock()

	return apps
}

// LastApplication returns the last registered Application.
// Handlers has access to the current Application,
// use `Context.Application()` instead.
func LastApplication() Application {
	mu.RLock()
	for i := len(registeredApps) - 1; i >= 0; i-- {
		if app := registeredApps[i]; app != nil {
			mu.RUnlock()
			return app
		}
	}
	mu.RUnlock()
	return nil
}

// GetApplication returns a registered Application
// based on its name. If the "appName" is not unique
// across Applications, then it will return the newest one.
func GetApplication(appName string) (Application, bool) {
	mu.RLock()

	for i := len(registeredApps) - 1; i >= 0; i-- {
		if app := registeredApps[i]; app != nil && app.String() == appName {
			mu.RUnlock()
			return app, true
		}
	}

	mu.RUnlock()
	return nil, false
}

// MustGetApplication same as `GetApplication` but it
// panics if "appName" is not a registered Application's name.
func MustGetApplication(appName string) Application {
	app, ok := GetApplication(appName)
	if !ok || app == nil {
		panic(appName + " is not a registered Application")
	}

	return app
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
