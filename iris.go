package iris

import (
	// std packages
	stdContext "context"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/kataras/golog"

	// context for the handlers
	"github.com/kataras/iris/context"
	// core packages, needed to build the application
	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/core/host"
	"github.com/kataras/iris/core/netutil"
	"github.com/kataras/iris/core/router"
	// handlerconv conversions
	"github.com/kataras/iris/core/handlerconv"
	// cache conversions
	"github.com/kataras/iris/cache"
	// view
	"github.com/kataras/iris/view"
	// middleware used in Default method
	requestLogger "github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
)

const (
	// Version is the current version number of the Iris Web Framework.
	Version = "8.2.5"
)

// HTTP status codes as registered with IANA.
// See: http://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
// Raw Copy from the net/http std package in order to recude the import path of "net/http" for the users.
//
// Copied from `net/http` package.
const (
	StatusContinue           = 100 // RFC 7231, 6.2.1
	StatusSwitchingProtocols = 101 // RFC 7231, 6.2.2
	StatusProcessing         = 102 // RFC 2518, 10.1

	StatusOK                   = 200 // RFC 7231, 6.3.1
	StatusCreated              = 201 // RFC 7231, 6.3.2
	StatusAccepted             = 202 // RFC 7231, 6.3.3
	StatusNonAuthoritativeInfo = 203 // RFC 7231, 6.3.4
	StatusNoContent            = 204 // RFC 7231, 6.3.5
	StatusResetContent         = 205 // RFC 7231, 6.3.6
	StatusPartialContent       = 206 // RFC 7233, 4.1
	StatusMultiStatus          = 207 // RFC 4918, 11.1
	StatusAlreadyReported      = 208 // RFC 5842, 7.1
	StatusIMUsed               = 226 // RFC 3229, 10.4.1

	StatusMultipleChoices   = 300 // RFC 7231, 6.4.1
	StatusMovedPermanently  = 301 // RFC 7231, 6.4.2
	StatusFound             = 302 // RFC 7231, 6.4.3
	StatusSeeOther          = 303 // RFC 7231, 6.4.4
	StatusNotModified       = 304 // RFC 7232, 4.1
	StatusUseProxy          = 305 // RFC 7231, 6.4.5
	_                       = 306 // RFC 7231, 6.4.6 (Unused)
	StatusTemporaryRedirect = 307 // RFC 7231, 6.4.7
	StatusPermanentRedirect = 308 // RFC 7538, 3

	StatusBadRequest                   = 400 // RFC 7231, 6.5.1
	StatusUnauthorized                 = 401 // RFC 7235, 3.1
	StatusPaymentRequired              = 402 // RFC 7231, 6.5.2
	StatusForbidden                    = 403 // RFC 7231, 6.5.3
	StatusNotFound                     = 404 // RFC 7231, 6.5.4
	StatusMethodNotAllowed             = 405 // RFC 7231, 6.5.5
	StatusNotAcceptable                = 406 // RFC 7231, 6.5.6
	StatusProxyAuthRequired            = 407 // RFC 7235, 3.2
	StatusRequestTimeout               = 408 // RFC 7231, 6.5.7
	StatusConflict                     = 409 // RFC 7231, 6.5.8
	StatusGone                         = 410 // RFC 7231, 6.5.9
	StatusLengthRequired               = 411 // RFC 7231, 6.5.10
	StatusPreconditionFailed           = 412 // RFC 7232, 4.2
	StatusRequestEntityTooLarge        = 413 // RFC 7231, 6.5.11
	StatusRequestURITooLong            = 414 // RFC 7231, 6.5.12
	StatusUnsupportedMediaType         = 415 // RFC 7231, 6.5.13
	StatusRequestedRangeNotSatisfiable = 416 // RFC 7233, 4.4
	StatusExpectationFailed            = 417 // RFC 7231, 6.5.14
	StatusTeapot                       = 418 // RFC 7168, 2.3.3
	StatusUnprocessableEntity          = 422 // RFC 4918, 11.2
	StatusLocked                       = 423 // RFC 4918, 11.3
	StatusFailedDependency             = 424 // RFC 4918, 11.4
	StatusUpgradeRequired              = 426 // RFC 7231, 6.5.15
	StatusPreconditionRequired         = 428 // RFC 6585, 3
	StatusTooManyRequests              = 429 // RFC 6585, 4
	StatusRequestHeaderFieldsTooLarge  = 431 // RFC 6585, 5
	StatusUnavailableForLegalReasons   = 451 // RFC 7725, 3

	StatusInternalServerError           = 500 // RFC 7231, 6.6.1
	StatusNotImplemented                = 501 // RFC 7231, 6.6.2
	StatusBadGateway                    = 502 // RFC 7231, 6.6.3
	StatusServiceUnavailable            = 503 // RFC 7231, 6.6.4
	StatusGatewayTimeout                = 504 // RFC 7231, 6.6.5
	StatusHTTPVersionNotSupported       = 505 // RFC 7231, 6.6.6
	StatusVariantAlsoNegotiates         = 506 // RFC 2295, 8.1
	StatusInsufficientStorage           = 507 // RFC 4918, 11.5
	StatusLoopDetected                  = 508 // RFC 5842, 7.2
	StatusNotExtended                   = 510 // RFC 2774, 7
	StatusNetworkAuthenticationRequired = 511 // RFC 6585, 6
)

// HTTP Methods copied from `net/http`.
const (
	MethodGet     = "GET"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodHead    = "HEAD"
	MethodPatch   = "PATCH"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
)

// MethodNone is an iris-specific "virtual" method
// to store the "offline" routes.
const MethodNone = "NONE"

// Application is responsible to manage the state of the application.
// It contains and handles all the necessary parts to create a fast web server.
type Application struct {
	// routing embedded | exposing APIBuilder's and Router's public API.
	*router.APIBuilder
	*router.Router
	ContextPool *context.Pool

	// config contains the configuration fields
	// all fields defaults to something that is working, developers don't have to set it.
	config *Configuration

	// the golog logger instance, defaults to "Info" level messages (all except "Debug")
	logger *golog.Logger

	// view engine
	view view.View
	// used for build
	once sync.Once

	mu sync.Mutex
	// Hosts contains a list of all servers (Host Supervisors) that this app is running on.
	//
	// Hosts may be empty only if application ran(`app.Run`) with `iris.Raw` option runner,
	// otherwise it contains a single host (`app.Hosts[0]`).
	//
	// Additional Host Supervisors can be added to that list by calling the `app.NewHost` manually.
	//
	// Hosts field is available after `Run` or `NewHost`.
	Hosts             []*host.Supervisor
	hostConfigurators []host.Configurator
}

// New creates and returns a fresh empty iris *Application instance.
func New() *Application {
	config := DefaultConfiguration()

	app := &Application{
		config:     &config,
		logger:     golog.Default,
		APIBuilder: router.NewAPIBuilder(),
		Router:     router.NewRouter(),
	}

	app.ContextPool = context.New(func() context.Context {
		return context.NewContext(app)
	})

	return app
}

// Default returns a new Application instance which, unlike `New`,
// recovers on panics and logs the incoming http requests.
func Default() *Application {
	app := New()
	app.Use(recover.New())
	app.Use(requestLogger.New())

	return app
}

// Configure can called when modifications to the framework instance needed.
// It accepts the framework instance
// and returns an error which if it's not nil it's printed to the logger.
// See configuration.go for more.
//
// Returns itself in order to be used like `app:= New().Configure(...)`
func (app *Application) Configure(configurators ...Configurator) *Application {
	for _, cfg := range configurators {
		cfg(app)
	}

	return app
}

// ConfigurationReadOnly returns an object which doesn't allow field writing.
func (app *Application) ConfigurationReadOnly() context.ConfigurationReadOnly {
	return app.config
}

// Logger returns the golog logger instance(pointer) that is being used inside the "app".
//
// Available levels:
// - "disable"
// - "fatal"
// - "error"
// - "warn"
// - "info"
// - "debug"
// Usage: app.Logger().SetLevel("error")
// Defaults to "info" level.
//
// Callers can use the application's logger which is
// the same `golog.Default` logger,
// to print custom logs too.
// Usage:
// app.Logger().Error/Errorf("...")
// app.Logger().Warn/Warnf("...")
// app.Logger().Info/Infof("...")
// app.Logger().Debug/Debugf("...")
//
// Setting one or more outputs: app.Logger().SetOutput(io.Writer...)
// Adding one or more outputs : app.Logger().AddOutput(io.Writer...)
//
// Adding custom levels requires import of the `github.com/kataras/golog` package:
//	First we create our level to a golog.Level
//	in order to be used in the Log functions.
//	var SuccessLevel golog.Level = 6
//	Register our level, just three fields.
//	golog.Levels[SuccessLevel] = &golog.LevelMetadata{
//		Name:    "success",
//		RawText: "[SUCC]",
//		// ColorfulText (Green Color[SUCC])
//		ColorfulText: "\x1b[32m[SUCC]\x1b[0m",
//	}
// Usage:
// app.Logger().SetLevel("success")
// app.Logger().Logf(SuccessLevel, "a custom leveled log message")
func (app *Application) Logger() *golog.Logger {
	return app.logger
}

var (
	// HTML view engine.
	// Conversion for the view.HTML.
	HTML = view.HTML
	// Django view engine.
	// Conversion for the view.Django.
	Django = view.Django
	// Handlebars view engine.
	// Conversion for the view.Handlebars.
	Handlebars = view.Handlebars
	// Pug view engine.
	// Conversion for the view.Pug.
	Pug = view.Pug
	// Amber view engine.
	// Conversion for the view.Amber.
	Amber = view.Amber
)

// NoLayout to disable layout for a particular template file
// A shortcut for the `view#NoLayout`.
const NoLayout = view.NoLayout

// RegisterView should be used to register view engines mapping to a root directory
// and the template file(s) extension.
func (app *Application) RegisterView(viewEngine view.Engine) {
	app.view.Register(viewEngine)
}

// View executes and writes the result of a template file to the writer.
//
// First parameter is the writer to write the parsed template.
// Second parameter is the relative, to templates directory, template filename, including extension.
// Third parameter is the layout, can be empty string.
// Forth parameter is the bindable data to the template, can be nil.
//
// Use context.View to render templates to the client instead.
// Returns an error on failure, otherwise nil.
func (app *Application) View(writer io.Writer, filename string, layout string, bindingData interface{}) error {
	if app.view.Len() == 0 {
		err := errors.New("view engine is missing, use `RegisterView`")
		app.Logger().Error(err)
		return err
	}

	err := app.view.ExecuteWriter(writer, filename, layout, bindingData)
	if err != nil {
		app.Logger().Error(err)
	}
	return err
}

var (
	// LimitRequestBodySize is a middleware which sets a request body size limit
	// for all next handlers in the chain.
	//
	// A shortcut for the `context#LimitRequestBodySize`.
	LimitRequestBodySize = context.LimitRequestBodySize
	// Gzip is a middleware which enables writing
	// using gzip compression, if client supports.
	//
	// A shortcut for the `context#Gzip`.
	Gzip = context.Gzip
	// FromStd converts native http.Handler, http.HandlerFunc & func(w, r, next) to context.Handler.
	//
	// Supported form types:
	// 		 .FromStd(h http.Handler)
	// 		 .FromStd(func(w http.ResponseWriter, r *http.Request))
	// 		 .FromStd(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc))
	//
	// A shortcut for the `handlerconv#FromStd`.
	FromStd = handlerconv.FromStd
	// Cache is a middleware providing cache functionalities
	// to the next handlers, can be used as: `app.Get("/", iris.Cache, aboutHandler)`.
	//
	// Examples can be found at: https://github.com/kataras/iris/tree/master/_examples/#caching
	Cache = cache.Handler
)

// SPA  accepts an "assetHandler" which can be the result of an
// app.StaticHandler or app.StaticEmbeddedHandler.
// It wraps the router and checks:
// if it;s an asset, if the request contains "." (this behavior can be changed via /core/router.NewSPABuilder),
// if the request is index, redirects back to the "/" in order to let the root handler to be executed,
// if it's not an asset then it executes the router, so the rest of registered routes can be
// executed without conflicts with the file server ("assetHandler").
//
// Use that instead of `StaticWeb` for root "/" file server.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/file-server/single-page-application
func (app *Application) SPA(assetHandler context.Handler) {
	s := router.NewSPABuilder(assetHandler)
	wrapper := s.BuildWrapper(app.ContextPool)
	app.Router.WrapRouter(wrapper)
}

// ConfigureHost accepts one or more `host#Configuration`, these configurators functions
// can access the host created by `app.Run`,
// they're being executed when application is ready to being served to the public.
//
// It's an alternative way to interact with a host that is automatically created by
// `app.Run`.
//
// These "configurators" can work side-by-side with the `iris#Addr, iris#Server, iris#TLS, iris#AutoTLS, iris#Listener`
// final arguments("hostConfigs") too.
//
// Note that these application's host "configurators" will be shared with the rest of
// the hosts that this app will may create (using `app.NewHost`), meaning that
// `app.NewHost` will execute these "configurators" everytime that is being called as well.
//
// These "configurators" should be registered before the `app.Run` or `host.Serve/Listen` functions.
func (app *Application) ConfigureHost(configurators ...host.Configurator) *Application {
	app.mu.Lock()
	app.hostConfigurators = append(app.hostConfigurators, configurators...)
	app.mu.Unlock()
	return app
}

// NewHost accepts a standar *http.Server object,
// completes the necessary missing parts of that "srv"
// and returns a new, ready-to-use, host (supervisor).
func (app *Application) NewHost(srv *http.Server) *host.Supervisor {
	app.mu.Lock()
	defer app.mu.Unlock()

	// set the server's handler to the framework's router
	if srv.Handler == nil {
		srv.Handler = app.Router
	}

	// check if different ErrorLog provided, if not bind it with the framework's logger
	if srv.ErrorLog == nil {
		srv.ErrorLog = log.New(app.logger.Printer.Output, "[HTTP Server] ", 0)
	}

	if srv.Addr == "" {
		srv.Addr = ":8080"
	}
	app.logger.Debugf("HTTP Server Addr: %s", srv.Addr)

	// create the new host supervisor
	// bind the constructed server and return it
	su := host.New(srv)

	if app.config.vhost == "" { // vhost now is useful for router subdomain on wildcard subdomains,
		// in order to correct decide what to do on:
		// mydomain.com -> invalid
		// localhost -> invalid
		// sub.mydomain.com -> valid
		// sub.localhost -> valid
		// we need the host (without port if 80 or 443) in order to validate these, so:
		app.config.vhost = netutil.ResolveVHost(srv.Addr)
	}

	app.logger.Debugf("VHost: %s", app.config.vhost)

	// the below schedules some tasks that will run among the server

	if !app.config.DisableStartupLog {
		// show the available info to exit from app.
		su.RegisterOnServe(host.WriteStartupLogOnServe(app.logger.Printer.Output)) // app.logger.Writer -> Info
		app.logger.Debugf("Host: Register startup notifier")
	}

	if !app.config.DisableInterruptHandler {
		// when CTRL+C/CMD+C pressed.
		shutdownTimeout := 5 * time.Second
		host.RegisterOnInterrupt(host.ShutdownOnInterrupt(su, shutdownTimeout))
		app.logger.Debugf("Host: Register server shutdown on interrupt(CTRL+C/CMD+C)")
	}

	su.IgnoredErrors = append(su.IgnoredErrors, app.config.IgnoreServerErrors...)
	app.logger.Debugf("Host: Server will ignore the following errors: %s", su.IgnoredErrors)
	su.Configure(app.hostConfigurators...)

	app.Hosts = append(app.Hosts, su)

	return su
}

// RegisterOnInterrupt registers a global function to call when CTRL+C/CMD+C pressed or a unix kill command received.
//
// A shortcut for the `host#RegisterOnInterrupt`.
var RegisterOnInterrupt = host.RegisterOnInterrupt

// Shutdown gracefully terminates all the application's server hosts.
// Returns an error on the first failure, otherwise nil.
func (app *Application) Shutdown(ctx stdContext.Context) error {
	for i, su := range app.Hosts {
		app.logger.Debugf("Host[%d]: Shutdown now", i)
		if err := su.Shutdown(ctx); err != nil {
			app.logger.Debugf("Host[%d]: Error while trying to shutdown", i)
			return err
		}
	}
	return nil
}

// Runner is just an interface which accepts the framework instance
// and returns an error.
//
// It can be used to register a custom runner with `Run` in order
// to set the framework's server listen action.
//
// Currently Runner is being used to declare the built'n server listeners.
//
// See `Run` for more.
type Runner func(*Application) error

// Listener can be used as an argument for the `Run` method.
// It can start a server with a custom net.Listener via server's `Serve`.
//
// Second argument is optional, it accepts one or more
// `func(*host.Configurator)` that are being executed
// on that specific host that this function will create to start the server.
// Via host configurators you can configure the back-end host supervisor,
// i.e to add events for shutdown, serve or error.
// An example of this use case can be found at:
// https://github.com/kataras/iris/blob/master/_examples/http-listening/notify-on-shutdown/main.go
// Look at the `ConfigureHost` too.
//
// See `Run` for more.
func Listener(l net.Listener, hostConfigs ...host.Configurator) Runner {
	return func(app *Application) error {
		app.config.vhost = netutil.ResolveVHost(l.Addr().String())
		return app.NewHost(new(http.Server)).
			Configure(hostConfigs...).
			Serve(l)
	}
}

// Server can be used as an argument for the `Run` method.
// It can start a server with a *http.Server.
//
// Second argument is optional, it accepts one or more
// `func(*host.Configurator)` that are being executed
// on that specific host that this function will create to start the server.
// Via host configurators you can configure the back-end host supervisor,
// i.e to add events for shutdown, serve or error.
// An example of this use case can be found at:
// https://github.com/kataras/iris/blob/master/_examples/http-listening/notify-on-shutdown/main.go
// Look at the `ConfigureHost` too.
//
// See `Run` for more.
func Server(srv *http.Server, hostConfigs ...host.Configurator) Runner {
	return func(app *Application) error {
		return app.NewHost(srv).
			Configure(hostConfigs...).
			ListenAndServe()
	}
}

// Addr can be used as an argument for the `Run` method.
// It accepts a host address which is used to build a server
// and a listener which listens on that host and port.
//
// Addr should have the form of [host]:port, i.e localhost:8080 or :8080.
//
// Second argument is optional, it accepts one or more
// `func(*host.Configurator)` that are being executed
// on that specific host that this function will create to start the server.
// Via host configurators you can configure the back-end host supervisor,
// i.e to add events for shutdown, serve or error.
// An example of this use case can be found at:
// https://github.com/kataras/iris/blob/master/_examples/http-listening/notify-on-shutdown/main.go
// Look at the `ConfigureHost` too.
//
// See `Run` for more.
func Addr(addr string, hostConfigs ...host.Configurator) Runner {
	return func(app *Application) error {
		return app.NewHost(&http.Server{Addr: addr}).
			Configure(hostConfigs...).
			ListenAndServe()
	}
}

// TLS can be used as an argument for the `Run` method.
// It will start the Application's secure server.
//
// Use it like you used to use the http.ListenAndServeTLS function.
//
// Addr should have the form of [host]:port, i.e localhost:443 or :443.
// CertFile & KeyFile should be filenames with their extensions.
//
// Second argument is optional, it accepts one or more
// `func(*host.Configurator)` that are being executed
// on that specific host that this function will create to start the server.
// Via host configurators you can configure the back-end host supervisor,
// i.e to add events for shutdown, serve or error.
// An example of this use case can be found at:
// https://github.com/kataras/iris/blob/master/_examples/http-listening/notify-on-shutdown/main.go
// Look at the `ConfigureHost` too.
//
// See `Run` for more.
func TLS(addr string, certFile, keyFile string, hostConfigs ...host.Configurator) Runner {
	return func(app *Application) error {
		return app.NewHost(&http.Server{Addr: addr}).
			Configure(hostConfigs...).
			ListenAndServeTLS(certFile, keyFile)
	}
}

// AutoTLS can be used as an argument for the `Run` method.
// It will start the Application's secure server using
// certifications created on the fly by the "autocert" golang/x package,
// so localhost may not be working, use it at "production" machine.
//
// Addr should have the form of [host]:port, i.e mydomain.com:443.
//
// Second argument is optional, it accepts one or more
// `func(*host.Configurator)` that are being executed
// on that specific host that this function will create to start the server.
// Via host configurators you can configure the back-end host supervisor,
// i.e to add events for shutdown, serve or error.
// An example of this use case can be found at:
// https://github.com/kataras/iris/blob/master/_examples/http-listening/notify-on-shutdown/main.go
// Look at the `ConfigureHost` too.
//
// See `Run` for more.
func AutoTLS(addr string, hostConfigs ...host.Configurator) Runner {
	return func(app *Application) error {
		return app.NewHost(&http.Server{Addr: addr}).
			Configure(hostConfigs...).
			ListenAndServeAutoTLS()
	}
}

// Raw can be used as an argument for the `Run` method.
// It accepts any (listen) function that returns an error,
// this function should be block and return an error
// only when the server exited or a fatal error caused.
//
// With this option you're not limited to the servers
// that iris can run by-default.
//
// See `Run` for more.
func Raw(f func() error) Runner {
	return func(app *Application) error {
		app.logger.Debugf("HTTP Server will start from unknown, external function")
		return f()
	}
}

// Build sets up, once, the framework.
// It builds the default router with its default macros
// and the template functions that are very-closed to iris.
func (app *Application) Build() error {
	rp := errors.NewReporter()

	app.once.Do(func() {
		rp.Describe("api builder: %v", app.APIBuilder.GetReport())

		if !app.Router.Downgraded() {
			// router
			// create the request handler, the default routing handler
			routerHandler := router.NewDefaultHandler()

			rp.Describe("router: %v", app.Router.BuildRouter(app.ContextPool, routerHandler, app.APIBuilder))
			// re-build of the router from outside can be done with;
			// app.RefreshRouter()
		}

		if app.view.Len() > 0 {
			app.logger.Debugf("%d registered view engine(s)", app.view.Len())
			// view engine
			// here is where we declare the closed-relative framework functions.
			// Each engine has their defaults, i.e yield,render,render_r,partial, params...
			rv := router.NewRoutePathReverser(app.APIBuilder)
			app.view.AddFunc("urlpath", rv.Path)
			// app.view.AddFunc("url", rv.URL)
			rp.Describe("view: %v", app.view.Load())
		}
	})

	return rp.Return()
}

// ErrServerClosed is returned by the Server's Serve, ServeTLS, ListenAndServe,
// and ListenAndServeTLS methods after a call to Shutdown or Close.
//
// A shortcut for the `http#ErrServerClosed`.
var ErrServerClosed = http.ErrServerClosed

// Run builds the framework and starts the desired `Runner` with or without configuration edits.
//
// Run should be called only once per Application instance, it blocks like http.Server.
//
// If more than one server needed to run on the same iris instance
// then create a new host and run it manually by `go NewHost(*http.Server).Serve/ListenAndServe` etc...
// or use an already created host:
// h := NewHost(*http.Server)
// Run(Raw(h.ListenAndServe), WithCharset("UTF-8"),
//	   						  WithRemoteAddrHeader("CF-Connecting-IP"),
//    						  WithoutServerError(iris.ErrServerClosed))
//
// The Application can go online with any type of server or iris's host with the help of
// the following runners:
// `Listener`, `Server`, `Addr`, `TLS`, `AutoTLS` and `Raw`.
func (app *Application) Run(serve Runner, withOrWithout ...Configurator) error {
	// first Build because it doesn't need anything from configuration,
	//  this give the user the chance to modify the router inside a configurator as well.
	if err := app.Build(); err != nil {
		return errors.PrintAndReturnErrors(err, app.logger.Errorf)
	}

	app.Configure(withOrWithout...)
	app.logger.Debugf("Application:  running using %d host(s)", len(app.Hosts)+1)

	if !app.config.DisableVersionChecker && app.logger.Level != golog.DisableLevel {
		go CheckVersion()
	}

	// this will block until an error(unless supervisor's DeferFlow called from a Task).
	err := serve(app)
	if err != nil {
		app.Logger().Error(err)
	}

	return err
}
