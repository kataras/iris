package iris

import (
	stdContext "context"
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/core/server"
	"github.com/kataras/iris/v14/core/server/closer"
	"github.com/kataras/iris/v14/internal/signalhandler"
	"github.com/kataras/iris/v14/middleware/requestid"

	"github.com/kataras/golog"
)

// Version is the current version of the Iris Web Framework.
const Version = "14.0.1"

// Byte unit helpers.
const (
	// B is the value of 1 byte.
	B = 1 << (10 * iota)
	// KB is the value of 1 kilobyte or 1024 bytes.
	KB
	// MB is the value of 1 megabyte or 1024 * 1024 bytes.
	MB
	// GB is the value of 1 gigabyte or 1024 * 1024 * 1024 bytes.
	GB
	// TB is the value of 1 terabyte or 1024 * 1024 * 1024 * 1024 bytes.
	TB
	// PB is the value of 1 petabyte or 1024 * 1024 * 1024 * 1024 * 1024 bytes.
	PB
	// EB is the value of 1 exabyte or 1024 * 1024 * 1024 * 1024 * 1024 * 1024 bytes.
	EB
)

// Application is responsible to manage the state of the application.
// It contains and handles all the necessary parts to create a fast web server.
type Application struct {
	// routing embedded | exposing RouterBuilder's and Router's public API.
	*router.RouterBuilder

	// options contains the configuration fields,
	// all fields are optional and can be set through the `Options` functional options.
	options *Options
	closers closer.Closers

	// shutdowns counts the Shutdown calls that are running and shutdownsDone is
	// closed when the last of them returns. Run waits on it: the server's own
	// shutdown is only the first closer of the list, so it returns while the
	// closers after it are still running. Both are guarded by shutdownsMu.
	shutdownsMu   sync.Mutex
	shutdowns     int
	shutdownsDone chan struct{}

	contextPool *context.Pool

	// the golog logger instance, defaults to "Info" level messages (all except "Debug")
	logger *golog.Logger

	// used for build
	built       bool
	defaultMode bool
	// OnBuild is a single function which
	// is fired on the first `Build` method call.
	// If reports an error then the execution
	// is stopped and the error is logged.
	// It's nil by default except when `Switch` instead of `New` or `Default`
	// is used to initialize the Application.
	// Users can wrap it to accept more events.
	OnBuild func() error

	// OnBuilt is a single function which is fired after the application is built.
	OnBuilt func() error
}

// New creates and returns a fresh empty iris *Application instance.
//
// Usage:
//
//	app := iris.New(iris.Options{...})
//	app.Get("/", func(ctx iris.Context) {...})
//	app.Listen(":8080")
func New(configurators ...Configurator) *Application {
	// Create an Iris Application instance with the default options,
	// so options can be further modified by the configurators below.
	defaultOptions := newDefaultOptions()
	app := &Application{
		options: defaultOptions,
	}
	app.RouterBuilder = router.New(app)
	// Set the logger to the default one.
	app.logger = golog.Default.Child(app)
	// Set the context error handler to the default one.
	app.contextPool = context.New(func() any {
		return context.NewContext(app)
	})

	// Now, it's time to allow further custom configuration.
	// Each configurator can either change the default options or the application's state (including registering handlers and middlewares).
	for _, configurator := range configurators {
		configurator.Configure(app)
	}

	// If at least one configurator changed the application's name, let's set it to the logger too.
	if app.options.Name != "" && app.logger.Prefix == "" {
		app.logger.SetChildPrefix(app.options.Name)
	}

	context.RegisterApplication(app)
	return app
}

// EditOptions allows the caller to change the application's options.
// Should be called before BuildRouter, Build, Listen or Run methods.
//
// The changes reach the root Router and every Router created afterwards. A
// child Router (Router, Subdomain) copies the context options when it is
// created and keeps that copy, so call EditOptions before registering any
// child Router if the change is an Options.Context one.
//
// The router options (Options.Router), EnableDynamicHandler included, are
// read at the first build. Changing them afterwards has no effect, also
// when BuildRouter runs again.
func (app *Application) EditOptions(fn func(*Options)) {
	if app.built {
		app.logger.Warn("EditOptions: application is already built, changes to the options may not be applied")
	}
	fn(app.options)
	app.Configure(app.options) // call configure in order to set the host and other options like request or/and response options middleware.
}

// NewContextWrapper returns a new ContextWrapper. It uses the default pool.
// The default pool's AcquireFunc returns a zero value of T.
// The default pool's ReleaseFunc does nothing.
//
// Usage:
//
// Initialize a new context wrapper using the default pool,
// where the myCustomContext SHOULD implement the `SetContext(ctx iris.Context)` method.
//
//	w := iris.NewContextWrapper[myCustomContext]()
//
// Use it to wrap the handlers.
//
//	app.Get("/", w.Handler(index))
//
// See the `context/NewContextWrapperWithPool` for more.
func NewContextWrapper[T any, contextPtr context.ContextSetterPtr[T]]() *context.ContextWrapper[contextPtr] {
	return context.NewContextWrapper[T, contextPtr]()
}

// BuildRouter builds the router, it's being called automatically by the `Run` & `Listen` methods.
//
// A second call returns an error, unless the application was built with
// Options.Router.EnableDynamicHandler: then it compiles the routes again,
// which is how routes are added or removed while the server runs.
func (app *Application) BuildRouter() error { // overrides the underline RouterBuilder's BuildRouter method to pass the app's Router options.
	return app.RouterBuilder.BuildRouter(app.options.Router, app.logger, app.contextPool)
}

// Default returns a new Application with a development-friendly setup already
// in place. Routes are registered on it exactly as on [New]: the returned
// application is NOT built yet, so anything registered after this call is
// served.
//
// It registers, in execution order:
//
//   - the "debug" logger level,
//   - the rest/errors path parameter type and context error handlers,
//
// and, as pre-routing middleware in this order:
//
//   - panic recovery, which reports the panic through rest/errors,
//   - CORS for every origin ("*"),
//   - the "Server: Iris" response header,
//   - request and response compression, bounded by compress.Safe,
//   - rest/errors.Handler, which renders a failure status nobody gave a body to,
//   - a request id per request (the requestid middleware).
//
// So a bare failure status (ctx.StopWithStatus(403), ctx.NotFound()) and an
// unknown path answer the rest/errors JSON envelope, as they do on a
// [NewBuilder] application. Register your own with HandleNoRoute or
// HandleNoMethod before the application is built to answer 404 or 405
// yourself.
//
// Use it while you explore, not in production: see [NewBuilder] for an
// application whose CORS, logging, health and error handling you choose.
//
// Usage:
//
//	app := iris.Default()
//	app.Get("/", func(ctx iris.Context) {...})
//	app.Listen(":8080")
func Default() *Application {
	defaultConfigurator := func(app *Application) {
		app.logger.SetLevel("debug")
		app.defaultMode = true
	}

	b := NewBuilder(ConfiguratorFunc(defaultConfigurator)).
		NoPrefix().
		AllowOrigin("*").
		Compression(true).
		LogRequests(false).
		NoHealth().
		Services().
		RouterMiddlewares(requestid.New()).
		Middlewares()

	return b.(*builder).unbuilt()
}

// String completes the fmt.Stringer interface.
// It returns the given application's name.
func (app *Application) String() string {
	return app.options.Name
}

// Configure can called when modifications to the framework instance needed.
// It accepts the framework instance
// and returns an error which if it's not nil it's printed to the logger.
// See Options and Configurator for what a configurator can change.
//
// Returns itself in order to be used like `app:= New().Configure(...)`
func (app *Application) Configure(configurators ...Configurator) *Application {
	for _, configurator := range configurators {
		if configurator == nil {
			continue
		}

		configurator.Configure(app)
	}

	return app
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
// Or set it through Options.LogLevel, either at construction or later with
// app.EditOptions. Defaults to "info" level.
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
// The same method exists as Context.IsDebug() too.
func (app *Application) IsDebug() bool {
	return app.logger.Level >= golog.DebugLevel
}

// ContextPool returns the Iris sync.Pool which holds the contexts values.
// Iris automatically releases the request context, so you don't have to use it.
// It's only useful to manually release the context on cases that connection
// is hijacked by a third-party middleware and the http handler return too fast.
func (app *Application) ContextPool() *context.Pool {
	return app.contextPool
}

// Shutdown gracefully terminates the application by running every registered closer,
// in order, passing the given context to each one. The running server's own Shutdown is
// one of those closers: `Run` prepends it to the list, so it is the first to be called.
//
// The method takes a context (ctx) which allows for timeout and cancellation capabilities.
// This is particularly useful when you want to control how long the shutdown process should wait
// before forcefully closing.
//
// `Run` and `Listen` return once this method has run the whole list, not when
// the server's own closer, the first of it, has returned. So a main that ends
// with `Listen` lets every closer finish. The http.Server that
// ServerOptions.OnServerCreated hands out is not this list: a `Shutdown` or
// `Close` called on it runs no closer and releases them the moment it is
// called, not when the requests in flight have been drained.
//
// A handler that stops the application calls Shutdown on another goroutine, with
// a context of its own: go app.Shutdown(context.Background()), or one that
// carries a deadline. Called in place, the server's closer waits for the request
// that called it. With a context that has no deadline, the handler's own Context
// included, it does not return and `Run` waits with it. With a deadline it
// returns at the deadline, after closing every connection that is left, the
// caller's own included. The handler's Context is no better on another
// goroutine: it is canceled when the handler returns, which closes the other
// connections at once.
//
// Errors from individual closers are accumulated with errors.Join, so each one can be
// matched through errors.Is or errors.As. It returns nil when every closer succeeds.
// Calling it before `Run` simply closes whatever else was registered through `AddCloser`.
//
// Usage:
//
//	app := iris.New()
//	[...]
//	app.Shutdown(context.Background())
func (app *Application) Shutdown(ctx stdContext.Context) error {
	defer app.beginShutdown()() // Run returns after this call does.

	return app.closers.CloseWithContext(ctx)
}

// beginShutdown counts a Shutdown call that is about to run and returns the
// function that marks its end.
func (app *Application) beginShutdown() (end func()) {
	app.shutdownsMu.Lock()
	if app.shutdowns == 0 {
		app.shutdownsDone = make(chan struct{})
	}
	app.shutdowns++
	app.shutdownsMu.Unlock()

	return func() {
		app.shutdownsMu.Lock()
		if app.shutdowns--; app.shutdowns == 0 {
			close(app.shutdownsDone)
		}
		app.shutdownsMu.Unlock()
	}
}

// waitShutdown blocks until no Shutdown call of this application is running. It
// returns at once when there is none.
func (app *Application) waitShutdown() {
	app.shutdownsMu.Lock()
	done := app.shutdownsDone
	running := app.shutdowns > 0
	app.shutdownsMu.Unlock()

	if running {
		<-done
	}
}

// RegisterOnBuild registers callback functions to be executed before and after the application is built.
// The beforeBuild callback runs at the very start of `Build`, before the logger level is set
// and before the router is built; afterBuild runs once the internal state is constructed.
// Both run before the server is started and an error returned by either one aborts the build.
func (app *Application) RegisterOnBuild(beforeBuild, afterBuild func() error) {
	if beforeBuild != nil {
		app.registerOnBuild(beforeBuild)
	}

	if afterBuild != nil {
		app.registerOnBuilt(afterBuild)
	}
}

func (app *Application) registerOnBuild(fn func() error) {
	if app.OnBuild == nil {
		app.OnBuild = fn
		return
	}

	old := app.OnBuild
	app.OnBuild = func() error {
		if err := old(); err != nil {
			return err
		}
		return fn()
	}
}

func (app *Application) registerOnBuilt(fn func() error) {
	if app.OnBuilt == nil {
		app.OnBuilt = fn
		return
	}

	old := app.OnBuilt
	app.OnBuilt = func() error {
		if err := old(); err != nil {
			return err
		}
		return fn()
	}
}

// Build prepares the Iris Application to run by constructing its internal state.
// This method is called automatically when the application's Run method is invoked,
// but it can also be called manually if needed.
//
// The Build process includes the following steps:
//
//  1. If the application has already been built, it returns immediately.
//
//  2. If an OnBuild callback is set, it is executed. If the callback returns an error,
//     the build process is halted and the error is returned.
//
//  3. The application's logger level is set based on the configuration options.
//
//  4. If the application is not in a downgraded state, the router is built through
//     BuildRouter with the application's Router options, logger and context pool.
//     Lowercase routing is enforced there when Options.Router.ForceLowercaseRouting is set.
//     A router that BuildRouter already built by hand is left as it is, unless
//     Options.Router.EnableDynamicHandler is set: then its routes are compiled again.
//
//  5. If an OnBuilt callback is set, it is executed. If the callback returns an error,
//     the build process is halted and the error is returned.
//
// The method returns an error if any step of the build process fails.
// After the method had invoked, the Application's Handler field is a
// compatible http.Handler that can be served by any standard http server.
//
// Build runs once. A second call returns nil without rebuilding, so routes
// registered after the first call are not served. This matters when the
// Application came from [NewBuilder], whose own Build already ran this one:
// register those routes through the builder's API step instead.
// To add or remove routes after the build, set
// Options.Router.EnableDynamicHandler before it and call BuildRouter again.
//
// Usage:
//
//	app := iris.New()
//	err := app.Build()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	w := httptest.NewRecorder()
//	app.Handler.ServeHTTP(w, req)
func (app *Application) Build() error {
	if app.built {
		return nil
	}

	if cb := app.OnBuild; cb != nil {
		if err := cb(); err != nil {
			return fmt.Errorf("build: %w", err)
		}
	}

	app.built = true // even if fails.

	// check if a prior app.Logger().SetLevel called and if not
	// then set the defined configuration's log level.
	if app.logger.Level == golog.InfoLevel /* the default level */ {
		app.logger.SetLevel(app.options.LogLevel)
	}

	// A router that was built by hand through BuildRouter is built once,
	// unless Options.Router.EnableDynamicHandler lets it be compiled again.
	routerBuilt := app.Handler != nil && app.ServeMux != nil && !app.options.Router.EnableDynamicHandler

	if !app.Downgraded() && !routerBuilt {
		// Create the request handler, the default routing handler.
		err := app.BuildRouter()
		if err != nil {
			return fmt.Errorf("build: router: %w", err)
		}
	}

	if cb := app.OnBuilt; cb != nil {
		if err := cb(); err != nil {
			return fmt.Errorf("built: %w", err)
		}
	}

	return nil
}

// AddCloser adds a function, or a structure which completes the io.Closer, which will be executed when the server is closed.
//
// [Application.Shutdown] runs it, after the server's own shutdown, and `Run`
// and `Listen` return once it has. So it must not wait for one of them to
// return: each would be waiting for the other. On an interrupt signal it runs
// only when Shutdown itself is wired to the signal with
// iris.RegisterOnInterrupt(app.Shutdown): the server's built-in handler drains
// the connections and knows nothing of this list.
func (app *Application) AddCloser(fn any) {
	app.closers.AddCloser(fn)
}

// Listen builds the application's components and starts the server on the specified TCP network address.
// The address must be in the form "host:port", where the host must be either a literal IP address or
// a hostname that can be resolved to IP addresses. The port must be a literal port number or a service name.
// If the host is a literal IPv6 address, it must be enclosed in square brackets, such as "[2001:db8::1]:80".
//
// If the Addr parameter is an empty string, the method defaults to ":http".
//
// The method sets various server options, including network type, keep-alive duration, socket sharding,
// and log writer before calling the Run method to start the server.
//
// Listen blocks until the server stops. After [Application.Shutdown] it
// returns nil, and only once that call has run its whole list of closers: the
// requests in flight are drained, or the deadline of the context is reached,
// and the closers that follow the server have returned by then. On an
// interrupt signal the callbacks of [RegisterOnInterrupt] and the server's own
// handler run in the order they were registered, so a callback registered
// before Listen runs first, while the server still serves. The handler drains
// the requests in flight for up to 15 seconds, and the process then exits, with
// status 0 unless a callback returned an error. Listen returns at that same
// moment, so the statements after it may not run: cleanup that has to happen on
// a signal goes to [RegisterOnInterrupt]. A failed start (a build
// error, or a listener that could not be opened) is logged and the process
// exits with status 1, so a program with nothing else to do calls Listen as a
// statement. Set [Options.DisableFatalRun] to have the error returned instead,
// for a process that runs more than one server or a test that opens a real
// listener.
//
// Usage:
//
//	app := iris.New()
//	app.Listen(":8080")
func (app *Application) Listen(addrHostPort string) error {
	return app.Run(ServerOptions{
		Addr:           addrHostPort,
		Network:        "tcp",
		KeepAlive:      3 * time.Minute,
		SocketSharding: true,
		LogWriter:      app.logger.Printer.TerminalOrStdout(),
	})
}

// Run starts the Iris Application with the provided server options.
// It performs several key steps to ensure the application is correctly set up before starting the server:
//
//  1. Builds the application's router and other necessary components.
//
//  2. Ensures that a log writer is set for the server. If not provided in the options,
//     it defaults to the application's logger output.
//
// 3. Prints the build information unless startup logs are explicitly disabled.
//
// 4. Resolves and sets the host for the application using the provided address if the host is not already set.
//
// 5. Configures the allowed hosts for the server, including subdomains, based on the application's routes.
//
//  6. Initializes a new server with the given options and sets up a listener to update the application's host
//     based on the final listener's URL.
//
// 7. Prepends the server's shutdown function to the application's list of closers for proper cleanup on termination.
//
// 8. Starts the HTTP(S) server and blocks until it stops.
//
// Run blocks until the server stops. After a graceful shutdown it returns nil,
// once [Application.Shutdown] has run its whole list and not when it begins: it
// waits for the requests in flight and for the closers that follow the server.
// A `Shutdown` or `Close` called on the http.Server of
// ServerOptions.OnServerCreated is not that list, and releases Run the moment
// it is called, see [Application.Shutdown]. On an interrupt signal the
// server drains for up to its Timeout.ShutdownOnInterruptTimeout, the callbacks
// of [RegisterOnInterrupt] receive a context with that deadline, and the
// process then exits, at the moment Run returns, so the statements after Run
// may not run. A failed
// start, or a server that stops with an error, is logged and the process exits
// with status 1; with [Options.DisableFatalRun] the error is logged at error
// level and returned to the caller instead. Either way it is reported at once:
// Run does not wait then for a Shutdown that is running its closers, nor for
// the callbacks of a signal. See [Application.Listen].
//
// Usage:
//
//	app := iris.New()
//	app.Run(iris.ServerOptions{Addr: ":8080"})
func (app *Application) Run(options ServerOptions) error {
	// build the router and the rest of the application's parts.
	if err := app.Build(); err != nil {
		return app.fail(err)
	}

	// ensure server log writer.
	if options.LogWriter == nil {
		options.LogWriter = app.logger.Printer.TerminalOrStdout()
	}

	// print the build information if startup logs are not disabled.
	if !options.DisableStartupLog {
		PrintBuildInfo(app.logger.Printer.TerminalOrStdout())
	}

	// set the host through given addr if missing.
	if app.options.Host == "" {
		app.options.SetHost(resolveRootHost(options.Addr))
	}

	// set the allowed hosts for the server, including subdomains.
	if rootHost := app.options.Host; rootHost != "" {
		if options.Addr == "" { // set the addr to the host if missing.
			options.Addr = rootHost
		}

		hosts := []string{rootHost}
		for _, subdomain := range app.RouterBuilder.Routes().ListSubdomains() {
			hosts = append(hosts, subdomain+rootHost)
		}

		options.PrependHostPatterns(hosts...)
	}

	// initialize the server.
	srv := server.NewServer(options)
	srv.OnListenerCreated(func(_ net.Listener, url *url.URL) error {
		app.options.SetHost(url.Host) // set the host forcely by the final listener's URL.
		return nil
	})
	// prepend server shutdown to the application's closers.
	app.closers.PrependCloser(srv.Shutdown)
	// start the http(s) server.
	if err := srv.ListenAndServe(app.Handler); err != nil {
		// A server that did not start, or that stopped with an error. Reported
		// first, before the waits below: the one for a signal ends with the
		// process, which exits with status 0, and would lose the error. The
		// price is that a Shutdown, or the callbacks of a signal, that are
		// running at this moment are not waited for: the fatal default exits
		// the process under them, as it did before Run waited for anything.
		return app.fail(err)
	}

	// The server's shutdown is the first closer of Shutdown, and ListenAndServe
	// returns with it: wait for the rest of the list before handing a main back
	// the right to end the process. Then for the callbacks of an interrupt
	// signal, which exits the process once they are done, so this is where Run
	// stops on a signal. In that order: a signal that arrives while the closers
	// are still running is waited for as well.
	app.waitShutdown()
	signalhandler.Wait()

	return nil
}

// fail reports a failed Run. By default it logs the error at fatal level,
// which exits the process with status 1. With Options.DisableFatalRun it logs
// at error level and hands the error back to the caller.
func (app *Application) fail(err error) error {
	if app.options.DisableFatalRun {
		app.logger.Error(err)
		return err
	}

	app.logger.Fatal(err)
	return err // not reached: Fatal exits the process.
}

// resolveRootHost determines the root host from the given address.
func resolveRootHost(addr string) string {
	if host, port, err := net.SplitHostPort(addr); err == nil {
		return formatHost(host, port)
	}
	return server.ResolveHostname(addr)
}

// formatHost formats the host and port into a proper address string.
func formatHost(host, port string) string {
	if host == "" {
		host = "localhost"
	}
	if port == "80" || port == "443" {
		return host
	}
	return host + ":" + port
}

// PrintBuildInfo prints the Iris web framework version and current program's build information.
func PrintBuildInfo(w io.Writer) {
	hasBuildInfo := BuildTime != "" && BuildRevision != ""
	tab := " "
	if hasBuildInfo {
		tab = "   "
	}
	fmt.Fprintf(w, "Iris Version:%s%s\n", tab, Version)

	if hasBuildInfo {
		fmt.Fprintf(w, "Build Time:     %s\nBuild Revision: %s\n", BuildTime, BuildRevision)
	}
	fmt.Fprintln(w)
}
