package iris

import (
	"bytes"
	stdContext "context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/host"
	"github.com/kataras/iris/v12/core/netutil"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/i18n"
	"github.com/kataras/iris/v12/middleware/cors"
	"github.com/kataras/iris/v12/middleware/recover"
	"github.com/kataras/iris/v12/middleware/requestid"
	"github.com/kataras/iris/v12/view"

	"github.com/kataras/golog"
	"github.com/kataras/tunnel"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
)

// Version is the current version of the Iris Web Framework.
const Version = "12.2.11"

// Byte unit helpers.
const (
	B = 1 << (10 * iota)
	KB
	MB
	GB
	TB
	PB
	EB
)

// Application is responsible to manage the state of the application.
// It contains and handles all the necessary parts to create a fast web server.
type Application struct {
	// routing embedded | exposing APIBuilder's and Router's public API.
	*router.APIBuilder
	*router.Router
	router.HTTPErrorHandler // if Router is Downgraded this is nil.
	ContextPool             *context.Pool
	// See SetContextErrorHandler, defaults to nil.
	contextErrorHandler context.ErrorHandler

	// config contains the configuration fields
	// all fields defaults to something that is working, developers don't have to set it.
	config *Configuration

	// the golog logger instance, defaults to "Info" level messages (all except "Debug")
	logger *golog.Logger

	// I18n contains localization and internationalization support.
	// Use the `Load` or `LoadAssets` to locale language files.
	//
	// See `Context#Tr` method for request-based translations.
	I18n *i18n.I18n

	// Validator is the request body validator, defaults to nil.
	Validator context.Validator
	// Minifier to minify responses.
	minifier *minify.M

	// view engine
	view *view.View
	// used for build
	builded     bool
	defaultMode bool
	// OnBuild is a single function which
	// is fired on the first `Build` method call.
	// If reports an error then the execution
	// is stopped and the error is logged.
	// It's nil by default except when `Switch` instead of `New` or `Default`
	// is used to initialize the Application.
	// Users can wrap it to accept more events.
	OnBuild func() error

	mu sync.RWMutex
	// name is the application name and the log prefix for
	// that Application instance's Logger. See `SetName` and `String`.
	// Defaults to IRIS_APP_NAME envrinoment variable otherwise empty.
	name string
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
	runError          error
	runErrorMu        sync.RWMutex
}

// New creates and returns a fresh empty iris *Application instance.
func New() *Application {
	config := DefaultConfiguration()
	app := &Application{
		config:   &config,
		Router:   router.NewRouter(),
		I18n:     i18n.New(),
		minifier: newMinifier(),
		view:     new(view.View),
	}

	logger := newLogger(app)
	app.logger = logger
	app.APIBuilder = router.NewAPIBuilder(logger)
	app.ContextPool = context.New(func() interface{} {
		return context.NewContext(app)
	})

	context.RegisterApplication(app)
	return app
}

// Default returns a new Application.
// Default with "debug" Logger Level.
// Localization enabled on "./locales" directory
// and HTML templates on "./views" or "./templates" directory.
// CORS (allow all), Recovery and
// Request ID middleware already registered.
func Default() *Application {
	app := New()
	// Set default log level.
	app.logger.SetLevel("debug")
	app.logger.Debugf(`Log level set to "debug"`)

	/* #2046.
	// Register the accesslog middleware.
	logFile, err := os.OpenFile("./access.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err == nil {
		// Close the file on shutdown.
		app.ConfigureHost(func(su *Supervisor) {
			su.RegisterOnShutdown(func() {
				logFile.Close()
			})
		})

		ac := accesslog.New(logFile)
		ac.AddOutput(app.logger.Printer)
		app.UseRouter(ac.Handler)
		app.logger.Debugf("Using <%s> to log requests", logFile.Name())
	}
	*/

	// Register the requestid middleware
	// before recover so current Context.GetID() contains the info on panic logs.
	app.UseRouter(requestid.New())
	app.logger.Debugf("Using <UUID4> to identify requests")

	// Register the recovery, after accesslog and recover,
	// before end-developer's middleware.
	app.UseRouter(recover.New())

	// Register CORS (allow any origin to pass through) middleware.
	app.UseRouter(cors.New().
		ExtractOriginFunc(cors.DefaultOriginExtractor).
		ReferrerPolicy(cors.NoReferrerWhenDowngrade).
		AllowOriginFunc(cors.AllowAnyOrigin).
		Handler())

	app.defaultMode = true

	return app
}

func newLogger(app *Application) *golog.Logger {
	logger := golog.Default.Child(app)
	if name := os.Getenv("IRIS_APP_NAME"); name != "" {
		app.name = name
		logger.SetChildPrefix(name)
	}

	return logger
}

// SetName sets a unique name to this Iris Application.
// It sets a child prefix for the current Application's Logger.
// Look `String` method too.
//
// It returns this Application.
func (app *Application) SetName(appName string) *Application {
	app.mu.Lock()
	defer app.mu.Unlock()

	if app.name == "" {
		app.logger.SetChildPrefix(appName)
	}
	app.name = appName

	return app
}

// String completes the fmt.Stringer interface and it returns
// the application's name.
// If name was not set by `SetName` or `IRIS_APP_NAME` environment variable
// then this will return an empty string.
func (app *Application) String() string {
	return app.name
}

// WWW creates and returns a "www." subdomain.
// The difference from `app.Subdomain("www")` or `app.Party("www.")` is that the `app.WWW()` method
// wraps the router so all http(s)://mydomain.com will be redirect to http(s)://www.mydomain.com.
// Other subdomains can be registered using the app: `sub := app.Subdomain("mysubdomain")`,
// child subdomains can be registered using the www := app.WWW(); www.Subdomain("wwwchildSubdomain").
func (app *Application) WWW() router.Party {
	return app.SubdomainRedirect(app, app.Subdomain("www"))
}

// SubdomainRedirect registers a router wrapper which
// redirects(StatusMovedPermanently) a (sub)domain to another subdomain or to the root domain as fast as possible,
// before the router's try to execute route's handler(s).
//
// It receives two arguments, they are the from and to/target locations,
// 'from' can be a wildcard subdomain as well (app.WildcardSubdomain())
// 'to' is not allowed to be a wildcard for obvious reasons,
// 'from' can be the root domain(app) when the 'to' is not the root domain and visa-versa.
//
// Usage:
// www := app.Subdomain("www") <- same as app.Party("www.")
// app.SubdomainRedirect(app, www)
// This will redirect all http(s)://mydomain.com/%anypath% to http(s)://www.mydomain.com/%anypath%.
//
// One or more subdomain redirects can be used to the same app instance.
//
// If you need more information about this implementation then you have to navigate through
// the `core/router#NewSubdomainRedirectWrapper` function instead.
//
// Example: https://github.com/kataras/iris/tree/main/_examples/routing/subdomains/redirect
func (app *Application) SubdomainRedirect(from, to router.Party) router.Party {
	sd := router.NewSubdomainRedirectWrapper(app.ConfigurationReadOnly().GetVHost, from.GetRelPath(), to.GetRelPath())
	app.Router.AddRouterWrapper(sd)
	return to
}

// Configure can called when modifications to the framework instance needed.
// It accepts the framework instance
// and returns an error which if it's not nil it's printed to the logger.
// See configuration.go for more.
//
// Returns itself in order to be used like `app:= New().Configure(...)`
func (app *Application) Configure(configurators ...Configurator) *Application {
	for _, cfg := range configurators {
		if cfg != nil {
			cfg(app)
		}
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
// Or set the level through Configurartion's LogLevel or WithLogLevel functional option.
// Defaults to "info" level.
//
// Callers can use the application's logger which is
// the same `golog.Default.LastChild()` logger,
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
//
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
//
// Usage:
// app.Logger().SetLevel("success")
// app.Logger().Logf(SuccessLevel, "a custom leveled log message")
func (app *Application) Logger() *golog.Logger {
	return app.logger
}

// IsDebug reports whether the application is running
// under debug/development mode.
// It's just a shortcut of Logger().Level >= golog.DebugLevel.
// The same method existss as Context.IsDebug() too.
func (app *Application) IsDebug() bool {
	return app.logger.Level >= golog.DebugLevel
}

// I18nReadOnly returns the i18n's read-only features.
// See `I18n` method for more.
func (app *Application) I18nReadOnly() context.I18nReadOnly {
	return app.I18n
}

// Validate validates a value and returns nil if passed or
// the failure reason if does not.
func (app *Application) Validate(v interface{}) error {
	if app.Validator == nil {
		return nil
	}

	// val := reflect.ValueOf(v)
	// if val.Kind() == reflect.Ptr && !val.IsNil() {
	// 	val = val.Elem()
	// }

	// if val.Kind() == reflect.Struct && val.Type() != timeType {
	// 	return app.Validator.Struct(v)
	// }

	// no need to check the kind, underline lib does it but in the future this may change (look above).
	err := app.Validator.Struct(v)
	if err != nil {
		if !strings.HasPrefix(err.Error(), "validator: ") {
			return err
		}
	}

	return nil
}

func newMinifier() *minify.M {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("image/svg+xml", svg.Minify)
	m.AddFuncRegexp(regexp.MustCompile("^(application|text)/(x-)?(java|ecma)script$"), js.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]json$"), json.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]xml$"), xml.Minify)
	return m
}

// Minify is a middleware which minifies the responses
// based on the response content type.
// Note that minification might be slower, caching is advised.
// Customize the minifier through `Application.Minifier()`.
// Usage:
// app.Use(iris.Minify)
func Minify(ctx Context) {
	w := ctx.Application().Minifier().ResponseWriter(ctx.ResponseWriter().Naive(), ctx.Request())
	// Note(@kataras):
	// We don't use defer w.Close()
	// because this response writer holds a sync.WaitGroup under the hoods
	// and we MUST be sure that its wg.Wait is called on request cancelation
	// and not in the end of handlers chain execution
	// (which if running a time-consuming task it will delay its resource release).
	ctx.OnCloseErr(w.Close)
	ctx.ResponseWriter().SetWriter(w)
	ctx.Next()
}

// Minifier returns the minifier instance.
// By default it can minifies:
// - text/html
// - text/css
// - image/svg+xml
// - application/text(javascript, ecmascript, json, xml).
// Use that instance to add custom Minifiers before server ran.
func (app *Application) Minifier() *minify.M {
	return app.minifier
}

// RegisterView registers a view engine for the application.
// Children can register their own too. If no Party view Engine is registered
// then this one will be used to render the templates instead.
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
	if !app.view.Registered() {
		err := errors.New("view engine is missing, use `RegisterView`")
		app.logger.Error(err)
		return err
	}

	return app.view.ExecuteWriter(writer, filename, layout, bindingData)
}

// GetContextPool returns the Iris sync.Pool which holds the contexts values.
// Iris automatically releases the request context, so you don't have to use it.
// It's only useful to manually release the context on cases that connection
// is hijacked by a third-party middleware and the http handler return too fast.
func (app *Application) GetContextPool() *context.Pool {
	return app.ContextPool
}

// SetContextErrorHandler can optionally register a handler to handle
// and fire a customized error body to the client on JSON write failures.
//
// ExampleCode:
//
//	 type contextErrorHandler struct{}
//	 func (e *contextErrorHandler) HandleContextError(ctx iris.Context, err error) {
//		 errors.HandleError(ctx, err)
//	 }
//	 ...
//	 app.SetContextErrorHandler(new(contextErrorHandler))
func (app *Application) SetContextErrorHandler(errHandler context.ErrorHandler) *Application {
	app.contextErrorHandler = errHandler
	return app
}

// GetContextErrorHandler returns the handler which handles errors
// on JSON write failures.
func (app *Application) GetContextErrorHandler() context.ErrorHandler {
	return app.contextErrorHandler
}

// ConfigureHost accepts one or more `host#Configuration`, these configurators functions
// can access the host created by `app.Run` or `app.Listen`,
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

const serverLoggerPrefix = "[HTTP Server] "

type customHostServerLogger struct { // see #1875
	parent     io.Writer
	ignoreLogs [][]byte
}

var newLineBytes = []byte("\n")

func newCustomHostServerLogger(w io.Writer, ignoreLogs []string) *customHostServerLogger {
	prefixAsByteSlice := []byte(serverLoggerPrefix)

	// build the ignore lines.
	ignoreLogsAsByteSlice := make([][]byte, 0, len(ignoreLogs))
	for _, s := range ignoreLogs {
		ignoreLogsAsByteSlice = append(ignoreLogsAsByteSlice, append(prefixAsByteSlice, []byte(s)...)) // append([]byte(s), newLineBytes...)
	}

	return &customHostServerLogger{
		parent:     w,
		ignoreLogs: ignoreLogsAsByteSlice,
	}
}

func (l *customHostServerLogger) Write(p []byte) (int, error) {
	for _, ignoredLogBytes := range l.ignoreLogs {
		if bytes.Equal(bytes.TrimSuffix(p, newLineBytes), ignoredLogBytes) {
			return 0, nil
		}
	}

	return l.parent.Write(p)
}

// this may change during parallel jobs (see Application.NonBlocking & Wait).
func (app *Application) getVHost() string {
	app.mu.RLock()
	vhost := app.config.VHost
	app.mu.RUnlock()
	return vhost
}

func (app *Application) setVHost(vhost string) {
	app.mu.Lock()
	app.config.VHost = vhost
	app.mu.Unlock()
}

// NewHost accepts a standard *http.Server object,
// completes the necessary missing parts of that "srv"
// and returns a new, ready-to-use, host (supervisor).
func (app *Application) NewHost(srv *http.Server) *host.Supervisor {
	if app.getVHost() == "" { // vhost now is useful for router subdomain on wildcard subdomains,
		// in order to correct decide what to do on:
		// mydomain.com -> invalid
		// localhost -> invalid
		// sub.mydomain.com -> valid
		// sub.localhost -> valid
		// we need the host (without port if 80 or 443) in order to validate these, so:
		app.setVHost(netutil.ResolveVHost(srv.Addr))
	} else {
		context.GetDomain = func(_ string) string { // #1886
			return app.config.VHost // GetVHost: here we don't need mutex protection as it's request-time and all modifications are already made.
		}
	} // before lock.

	app.mu.Lock()
	defer app.mu.Unlock()

	// set the server's handler to the framework's router
	if srv.Handler == nil {
		srv.Handler = app.Router
	}

	// check if different ErrorLog provided, if not bind it with the framework's logger.
	if srv.ErrorLog == nil {
		serverLogger := newCustomHostServerLogger(app.logger.Printer.Output, app.config.IgnoreServerErrors)
		srv.ErrorLog = log.New(serverLogger, serverLoggerPrefix, 0)
	}

	if addr := srv.Addr; addr == "" {
		addr = ":8080"
		if len(app.Hosts) > 0 {
			if v := app.Hosts[0].Server.Addr; v != "" {
				addr = v
			}
		}

		srv.Addr = addr
	}

	// app.logger.Debugf("Host: addr is %s", srv.Addr)

	// create the new host supervisor
	// bind the constructed server and return it
	su := host.New(srv)
	// app.logger.Debugf("Host: virtual host is %s", app.config.VHost)

	// the below schedules some tasks that will run among the server

	if !app.config.DisableStartupLog {
		printer := app.logger.Printer.Output
		hostPrinter := host.WriteStartupLogOnServe(printer)
		if len(app.Hosts) == 0 { // print the version info on the first running host.
			su.RegisterOnServe(func(h host.TaskHost) {
				hasBuildInfo := BuildTime != "" && BuildRevision != ""
				tab := " "
				if hasBuildInfo {
					tab = "   "
				}
				fmt.Fprintf(printer, "Iris Version:%s%s\n", tab, Version)

				if hasBuildInfo {
					fmt.Fprintf(printer, "Build Time:     %s\nBuild Revision: %s\n", BuildTime, BuildRevision)
				}
				fmt.Fprintln(printer)

				hostPrinter(h)
			})
		} else {
			su.RegisterOnServe(hostPrinter)
		}

		// app.logger.Debugf("Host: register startup notifier")
	}

	if !app.config.DisableInterruptHandler {
		// when CTRL/CMD+C pressed.
		shutdownTimeout := 10 * time.Second
		RegisterOnInterrupt(host.ShutdownOnInterrupt(su, shutdownTimeout))
		// app.logger.Debugf("Host: register server shutdown on interrupt(CTRL+C/CMD+C)")
	}

	su.IgnoredErrors = append(su.IgnoredErrors, app.config.IgnoreServerErrors...)
	if len(su.IgnoredErrors) > 0 {
		app.logger.Debugf("Host: server will ignore the following errors: %s", su.IgnoredErrors)
	}

	su.Configure(app.hostConfigurators...)

	app.Hosts = append(app.Hosts, su)

	return su
}

// func (app *Application) OnShutdown(closers ...func()) {
// 	for _,cb := range closers {
// 		if cb == nil {
// 			continue
// 		}
// 		RegisterOnInterrupt(cb)
// 	}
// }

// Shutdown gracefully terminates all the application's server hosts and any tunnels.
// Returns an error on the first failure, otherwise nil.
func (app *Application) Shutdown(ctx stdContext.Context) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	defer app.setRunError(ErrServerClosed) // make sure to set the error so any .Wait calls return.

	for i, su := range app.Hosts {
		app.logger.Debugf("Host[%d]: Shutdown now", i)
		if err := su.Shutdown(ctx); err != nil {
			app.logger.Debugf("Host[%d]: Error while trying to shutdown", i)
			return err
		}
	}

	for _, t := range app.config.Tunneling.Tunnels {
		if t.Name == "" {
			continue
		}

		if err := app.config.Tunneling.StopTunnel(t); err != nil {
			return err
		}
	}

	return nil
}

// Build sets up, once, the framework.
// It builds the default router with its default macros
// and the template functions that are very-closed to iris.
//
// If error occurred while building the Application, the returns type of error will be an *errgroup.Group
// which let the callers to inspect the errors and cause, usage:
//
// import "github.com/kataras/iris/v12/core/errgroup"
//
//	errgroup.Walk(app.Build(), func(typ interface{}, err error) {
//		app.Logger().Errorf("%s: %s", typ, err)
//	})
func (app *Application) Build() error {
	if app.builded {
		return nil
	}

	if cb := app.OnBuild; cb != nil {
		if err := cb(); err != nil {
			return fmt.Errorf("build: %w", err)
		}
	}

	// start := time.Now()
	app.builded = true // even if fails.

	// check if a prior app.Logger().SetLevel called and if not
	// then set the defined configuration's log level.
	if app.logger.Level == golog.InfoLevel /* the default level */ {
		app.logger.SetLevel(app.config.LogLevel)
	}

	if app.defaultMode { // the app.I18n and app.View will be not available until Build.
		if !app.I18n.Loaded() {
			for _, s := range []string{"./locales/*/*", "./locales/*", "./translations"} {
				if _, err := os.Stat(s); err != nil {
					continue
				}

				if err := app.I18n.Load(s); err != nil {
					continue
				}

				app.I18n.SetDefault("en-US")
				break
			}
		}

		if !app.view.Registered() {
			for _, s := range []string{"./views", "./templates", "./web/views"} {
				if _, err := os.Stat(s); err != nil {
					continue
				}

				app.RegisterView(HTML(s, ".html"))
				break
			}
		}
	}

	if app.I18n.Loaded() {
		// {{ tr "lang" "key" arg1 arg2 }}
		app.view.AddFunc("tr", app.I18n.Tr)
		app.Router.PrependRouterWrapper(app.I18n.Wrapper())
	}

	if app.view.Registered() {
		app.logger.Debugf("Application: view engine %q is registered", app.view.Name())
		// view engine
		// here is where we declare the closed-relative framework functions.
		// Each engine has their defaults, i.e yield,render,render_r,partial, params...
		rv := router.NewRoutePathReverser(app.APIBuilder)
		app.view.AddFunc("urlpath", rv.Path)
		// app.view.AddFunc("url", rv.URL)
		if err := app.view.Load(); err != nil {
			return fmt.Errorf("build: view engine: %v", err)
		}
	}

	if !app.Router.Downgraded() {
		// router
		if _, err := injectLiveReload(app); err != nil {
			return fmt.Errorf("build: inject live reload: failed: %v", err)
		}

		if app.config.ForceLowercaseRouting {
			// This should always be executed first.
			app.Router.PrependRouterWrapper(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
				r.Host = strings.ToLower(r.Host)
				r.URL.Host = strings.ToLower(r.URL.Host)
				r.URL.Path = strings.ToLower(r.URL.Path)
				next(w, r)
			})
		}

		// create the request handler, the default routing handler
		routerHandler := router.NewDefaultHandler(app.config, app.logger)
		err := app.Router.BuildRouter(app.ContextPool, routerHandler, app.APIBuilder, false)
		if err != nil {
			return fmt.Errorf("build: router: %w", err)
		}
		app.HTTPErrorHandler = routerHandler

		if app.config.Timeout > 0 {
			app.Router.SetTimeoutHandler(app.config.Timeout, app.config.TimeoutMessage)

			app.ConfigureHost(func(su *Supervisor) {
				if su.Server.ReadHeaderTimeout == 0 {
					su.Server.ReadHeaderTimeout = app.config.Timeout + 5*time.Second
				}

				if su.Server.ReadTimeout == 0 {
					su.Server.ReadTimeout = app.config.Timeout + 10*time.Second
				}

				if su.Server.WriteTimeout == 0 {
					su.Server.WriteTimeout = app.config.Timeout + 15*time.Second
				}

				if su.Server.IdleTimeout == 0 {
					su.Server.IdleTimeout = app.config.Timeout + 25*time.Second
				}
			})
		}

		// re-build of the router from outside can be done with
		// app.RefreshRouter()
	}

	// if end := time.Since(start); end.Seconds() > 5 {
	// app.logger.Debugf("Application: build took %s", time.Since(start))

	return nil
}

// Runner is just an interface which accepts the framework instance
// and returns an error.
//
// It can be used to register a custom runner with `Run` in order
// to set the framework's server listen action.
//
// Currently `Runner` is being used to declare the builtin server listeners.
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
// https://github.com/kataras/iris/blob/main/_examples/http-server/notify-on-shutdown/main.go
// Look at the `ConfigureHost` too.
//
// See `Run` for more.
func Listener(l net.Listener, hostConfigs ...host.Configurator) Runner {
	return func(app *Application) error {
		app.config.SetVHost(netutil.ResolveVHost(l.Addr().String()))
		return app.NewHost(&http.Server{Addr: l.Addr().String()}).
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
// https://github.com/kataras/iris/blob/main/_examples/http-server/notify-on-shutdown/main.go
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
// https://github.com/kataras/iris/blob/main/_examples/http-server/notify-on-shutdown/main.go
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

var (
	// TLSNoRedirect is a `host.Configurator` which can be passed as last argument
	// to the `TLS` runner function. It disables the automatic
	// registration of redirection from "http://" to "https://" requests.
	// Applies only to the `TLS` runner.
	// See `AutoTLSNoRedirect` to register a custom fallback server for `AutoTLS` runner.
	TLSNoRedirect = func(su *host.Supervisor) { su.NoRedirect() }
	// AutoTLSNoRedirect is a `host.Configurator`.
	// It registers a fallback HTTP/1.1 server for the `AutoTLS` one.
	// The function accepts the letsencrypt wrapper and it
	// should return a valid instance of http.Server which its handler should be the result
	// of the "acmeHandler" wrapper.
	// Usage:
	//	 getServer := func(acme func(http.Handler) http.Handler) *http.Server {
	//	     srv := &http.Server{Handler: acme(yourCustomHandler), ...otherOptions}
	//	     go srv.ListenAndServe()
	//	     return srv
	//   }
	//   app.Run(iris.AutoTLS(":443", "example.com example2.com", "mail@example.com", getServer))
	//
	// Note that if Server.Handler is nil then the server is automatically ran
	// by the framework and the handler set to automatic redirection, it's still
	// a valid option when the caller wants just to customize the server's fields (except Addr).
	// With this host configurator the caller can customize the server
	// that letsencrypt relies to perform the challenge.
	// LetsEncrypt Certification Manager relies on http://example.com/.well-known/acme-challenge/<TOKEN>.
	AutoTLSNoRedirect = func(getFallbackServer func(acmeHandler func(fallback http.Handler) http.Handler) *http.Server) host.Configurator {
		return func(su *host.Supervisor) {
			su.NoRedirect()
			su.Fallback = getFallbackServer
		}
	}
)

// TLS can be used as an argument for the `Run` method.
// It will start the Application's secure server.
//
// Use it like you used to use the http.ListenAndServeTLS function.
//
// Addr should have the form of [host]:port, i.e localhost:443 or :443.
// "certFileOrContents" & "keyFileOrContents" should be filenames with their extensions
// or raw contents of the certificate and the private key.
//
// Last argument is optional, it accepts one or more
// `func(*host.Configurator)` that are being executed
// on that specific host that this function will create to start the server.
// Via host configurators you can configure the back-end host supervisor,
// i.e to add events for shutdown, serve or error.
// An example of this use case can be found at:
// https://github.com/kataras/iris/blob/main/_examples/http-server/notify-on-shutdown/main.go
// Look at the `ConfigureHost` too.
//
// See `Run` for more.
func TLS(addr string, certFileOrContents, keyFileOrContents string, hostConfigs ...host.Configurator) Runner {
	return func(app *Application) error {
		return app.NewHost(&http.Server{Addr: addr}).
			Configure(hostConfigs...).
			ListenAndServeTLS(certFileOrContents, keyFileOrContents)
	}
}

// AutoTLS can be used as an argument for the `Run` method.
// It will start the Application's secure server using
// certifications created on the fly by the "autocert" golang/x package,
// so localhost may not be working, use it at "production" machine.
//
// Addr should have the form of [host]:port, i.e mydomain.com:443.
//
// The whitelisted domains are separated by whitespace in "domain" argument,
// i.e "iris-go.com", can be different than "addr".
// If empty, all hosts are currently allowed. This is not recommended,
// as it opens a potential attack where clients connect to a server
// by IP address and pretend to be asking for an incorrect host name.
// Manager will attempt to obtain a certificate for that host, incorrectly,
// eventually reaching the CA's rate limit for certificate requests
// and making it impossible to obtain actual certificates.
//
// For an "e-mail" use a non-public one, letsencrypt needs that for your own security.
//
// Note: `AutoTLS` will start a new server for you
// which will redirect all http versions to their https, including subdomains as well.
//
// Last argument is optional, it accepts one or more
// `func(*host.Configurator)` that are being executed
// on that specific host that this function will create to start the server.
// Via host configurators you can configure the back-end host supervisor,
// i.e to add events for shutdown, serve or error.
// An example of this use case can be found at:
// https://github.com/kataras/iris/blob/main/_examples/http-server/notify-on-shutdown/main.go
// Look at the `ConfigureHost` too.
//
// Usage:
// app.Run(iris.AutoTLS("iris-go.com:443", "iris-go.com www.iris-go.com", "mail@example.com"))
//
// See `Run` and `core/host/Supervisor#ListenAndServeAutoTLS` for more.
func AutoTLS(
	addr string,
	domain string, email string,
	hostConfigs ...host.Configurator,
) Runner {
	return func(app *Application) error {
		return app.NewHost(&http.Server{Addr: addr}).
			Configure(hostConfigs...).
			ListenAndServeAutoTLS(domain, email, "letscache")
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

var (
	// ErrServerClosed is logged by the standard net/http server when the server is terminated.
	// Ignore it by passing this error to the `iris.WithoutServerError` configurator
	// on `Application.Run/Listen` method.
	//
	// An alias of the `http#ErrServerClosed`.
	ErrServerClosed = http.ErrServerClosed

	// ErrURLQuerySemicolon is logged by the standard net/http server when
	// the request contains a semicolon (;) wihch, after go1.17 it's not used as a key-value separator character.
	//
	// Ignore it by passing this error to the `iris.WithoutServerError` configurator
	// on `Application.Run/Listen` method.
	//
	// An alias of the `http#ErrServerClosed`.
	ErrURLQuerySemicolon = errors.New("http: URL query contains semicolon, which is no longer a supported separator; parts of the query may be stripped when parsed; see golang.org/issue/25192")
)

// Listen builds the application and starts the server
// on the TCP network address "host:port" which
// handles requests on incoming connections.
//
// Listen always returns a non-nil error except
// when NonBlocking option is being passed, so the error goes to the Wait method.
// Ignore specific errors by using an `iris.WithoutServerError(iris.ErrServerClosed)`
// as a second input argument.
//
// Listen is a shortcut of `app.Run(iris.Addr(hostPort, withOrWithout...))`.
// See `Run` for details.
func (app *Application) Listen(hostPort string, withOrWithout ...Configurator) error {
	return app.Run(Addr(hostPort), withOrWithout...)
}

// Run builds the framework and starts the desired `Runner` with or without configuration edits.
//
// Run should be called only once per Application instance, it blocks like http.Server.
//
// If more than one server needed to run on the same iris instance
// then create a new host and run it manually by `go NewHost(*http.Server).Serve/ListenAndServe` etc...
// or use an already created host:
// h := NewHost(*http.Server)
// Run(Raw(h.ListenAndServe), WithCharset("utf-8"), WithRemoteAddrHeader("CF-Connecting-IP"))
//
// The Application can go online with any type of server or iris's host with the help of
// the following runners:
// `Listener`, `Server`, `Addr`, `TLS`, `AutoTLS` and `Raw`.
func (app *Application) Run(serve Runner, withOrWithout ...Configurator) error {
	app.Configure(withOrWithout...)

	if err := app.Build(); err != nil {
		app.logger.Error(err)
		return err
	}

	app.ConfigureHost(func(host *Supervisor) {
		host.SocketSharding = app.config.SocketSharding
		host.KeepAlive = app.config.KeepAlive
	})

	app.tryStartTunneling()

	if len(app.Hosts) > 0 {
		app.logger.Debugf("Application: running using %d host(s)", len(app.Hosts)+1 /* +1 the current */)
	}

	if app.config.NonBlocking {
		go func() {
			err := app.serve(serve)
			if err != nil {
				app.setRunError(err)
			}
		}()

		return nil
	}

	// this will block until an error(unless supervisor's DeferFlow called from a Task)
	// or NonBlocking was passed (see above).
	return app.serve(serve)
}

func (app *Application) serve(serve Runner) error {
	err := serve(app)
	if err != nil {
		app.logger.Error(err)
	}
	return err
}

func (app *Application) setRunError(err error) {
	app.runErrorMu.Lock()
	app.runError = err
	app.runErrorMu.Unlock()
}

func (app *Application) getRunError() error {
	app.runErrorMu.RLock()
	err := app.runError
	app.runErrorMu.RUnlock()
	return err
}

// Wait blocks the main goroutine until the server application is up and running.
// Useful only when `Run` is called with `iris.NonBlocking()` option.
func (app *Application) Wait(ctx stdContext.Context) error {
	if !app.config.NonBlocking {
		return nil
	}

	// First check if there is an error already from the app.Run.
	if err := app.getRunError(); err != nil {
		return err
	}

	// Set the base for exponential backoff.
	base := 2.0

	// Get the maximum number of retries by context or force to 7 retries.
	var maxRetries int
	// Get the deadline of the context.
	if deadline, ok := ctx.Deadline(); ok {
		now := time.Now()
		timeout := deadline.Sub(now)

		maxRetries = getMaxRetries(timeout, base)
	} else {
		maxRetries = 7 // 256 seconds max.
	}

	// Set the initial retry interval.
	retryInterval := time.Second

	return app.tryConnect(ctx, maxRetries, retryInterval, base)
}

// getMaxRetries calculates the maximum number of retries from the retry interval and the base.
func getMaxRetries(retryInterval time.Duration, base float64) int {
	// Convert the retry interval to seconds.
	seconds := retryInterval.Seconds()
	// Apply the inverse formula.
	retries := math.Log(seconds)/math.Log(base) - 1
	return int(math.Round(retries))
}

// tryConnect tries to connect to the server with the given context and retry parameters.
func (app *Application) tryConnect(ctx stdContext.Context, maxRetries int, retryInterval time.Duration, base float64) error {
	// Try to connect to the server in a loop.
	for i := 0; i < maxRetries; i++ {
		// Check the context before each attempt.
		select {
		case <-ctx.Done():
			// Context is canceled, return the context error.
			return ctx.Err()
		default:
			address := app.getVHost() // Get this server's listening address.
			if address == "" {
				i-- // Note that this may be modified at another go routine of the serve method. So it may be empty at first chance. So retry fetching the VHost every 1 second.
				time.Sleep(time.Second)
				continue
			}

			// Context is not canceled, proceed with the attempt.
			conn, err := net.Dial("tcp", address)
			if err == nil {
				// Connection successful, close the connection and return nil.
				conn.Close()
				return nil // exit.
			} // ignore error.

			// Connection failed, wait for the retry interval and try again.
			time.Sleep(retryInterval)
			// After each failed attempt, check the server Run's error again.
			if err := app.getRunError(); err != nil {
				return err
			}

			// Increase the retry interval by the base raised to the power of the number of attempts.
			/*
				0	2 seconds
				1	4 seconds
				2	8 seconds
				3	~16 seconds
				4	~32 seconds
				5	~64 seconds
				6	~128 seconds
				7	~256 seconds
				8	~512 seconds
				...
			*/
			retryInterval = time.Duration(math.Pow(base, float64(i+1))) * time.Second
		}
	}
	// All attempts failed, return an error.
	return fmt.Errorf("failed to connect to the server after %d retries", maxRetries)
}

// https://ngrok.com/docs
func (app *Application) tryStartTunneling() {
	if len(app.config.Tunneling.Tunnels) == 0 {
		return
	}

	app.ConfigureHost(func(su *host.Supervisor) {
		su.RegisterOnServe(func(h host.TaskHost) {
			publicAddrs, err := tunnel.Start(app.config.Tunneling)
			if err != nil {
				app.logger.Errorf("Host: tunneling error: %v", err)
				return
			}

			publicAddr := publicAddrs[0]
			// to make subdomains resolution still based on this new remote, public addresses.
			app.setVHost(publicAddr[strings.Index(publicAddr, "://")+3:])

			directLog := []byte(fmt.Sprintf("• Public Address: %s\n", publicAddr))
			app.logger.Printer.Write(directLog) // nolint:errcheck
		})
	})
}
