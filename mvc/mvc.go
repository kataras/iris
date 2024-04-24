package mvc

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/hero"
	"github.com/kataras/iris/v12/websocket"

	"github.com/kataras/golog"
	"github.com/kataras/pio"
)

// Application is the high-level component of the "mvc" package.
// It's the API that you will be using to register controllers among with their
// dependencies that your controllers may expecting.
// It contains the Router(iris.Party) in order to be able to register
// template layout, middleware, done handlers as you used with the
// standard Iris APIBuilder.
//
// The Engine is created by the `New` method and it's the dependencies holder
// and controllers factory.
//
// See `mvc#New` for more.
type Application struct {
	container *hero.Container
	// This Application's Name. Keep names unique to each other.
	Name string

	Router               router.Party
	Controllers          []*ControllerActivator
	websocketControllers []websocket.ConnHandler

	// Disables verbose logging for controllers under this and its children mvc apps.
	// Defaults to false.
	controllersNoLog bool

	// Set custom path
	customPathWordFunc CustomPathWordFunc
}

func newApp(subRouter router.Party, container *hero.Container) *Application {
	app := &Application{
		Router:    subRouter,
		container: container,
	}

	// Register this Application so any field or method's input argument of
	// *mvc.Application can point to the current MVC application that the controller runs on.
	registerBuiltinDependencies(container, app)
	return app
}

// See `hero.BuiltinDependencies` too, here we are registering dependencies per MVC Application.
func registerBuiltinDependencies(container *hero.Container, deps ...interface{}) {
	for _, dep := range deps {
		depTyp := reflect.TypeOf(dep)
		for i, dependency := range container.Dependencies {
			if dependency.Static {
				if dependency.DestType == depTyp {
					// Remove any existing before register this one (see app.Clone).
					copy(container.Dependencies[i:], container.Dependencies[i+1:])
					container.Dependencies = container.Dependencies[:len(container.Dependencies)-1]
					break
				}
			}
		}

		container.Register(dep)
	}
}

// New returns a new mvc Application based on a "party".
// Application creates a new engine which is responsible for binding the dependencies
// and creating and activating the app's controller(s).
//
// Example: `New(app.Party("/todo"))` or `New(app)` as it's the same as `New(app.Party("/"))`.
func New(party router.Party) *Application {
	return newApp(party, party.ConfigureContainer().Container.Clone())
}

// Configure creates a new controller and configures it,
// this function simply calls the `New(party)` and its `.Configure(configurators...)`.
//
// A call of `mvc.New(app.Party("/path").Configure(buildMyMVC)` is equal to
//
//	`mvc.Configure(app.Party("/path"), buildMyMVC)`.
//
// Read more at `New() Application` and `Application#Configure` methods.
func Configure(party router.Party, configurators ...func(*Application)) *Application {
	// Author's Notes->
	// About the Configure's comment: +5 space to be shown in equal width to the previous or after line.
	//
	// About the Configure's design chosen:
	// Yes, we could just have a `New(party, configurators...)`
	// but I think the `New()` and `Configure(configurators...)` API seems more native to programmers,
	// at least to me and the people I ask for their opinion between them.
	// Because the `New()` can actually return something that can be fully configured without its `Configure`,
	// its `Configure` is there just to design the apps better and help end-devs to split their code wisely.
	return New(party).Configure(configurators...)
}

// Configure can be used to pass one or more functions that accept this
// Application, use this to add dependencies and controller(s).
//
// Example: `New(app.Party("/todo")).Configure(func(mvcApp *mvc.Application){...})`.
func (app *Application) Configure(configurators ...func(*Application)) *Application {
	for _, c := range configurators {
		c(app)
	}
	return app
}

// SetName sets a unique name to this MVC Application.
// Used for logging, not used in runtime yet, but maybe useful for future features.
//
// It returns this Application.
func (app *Application) SetName(appName string) *Application {
	app.Name = appName
	return app
}

// SetCustomPathWordFunc sets a custom function
// which is responsible to override the existing controllers method parsing.
func (app *Application) SetCustomPathWordFunc(wordFunc CustomPathWordFunc) *Application {
	app.customPathWordFunc = wordFunc
	return app
}

// SetControllersNoLog disables verbose logging for next registered controllers
// under this App and its children of `Application.Party` or `Application.Clone`.
//
// To disable logging for routes under a Party,
// see `Party.SetRoutesNoLog` instead.
//
// Defaults to false when log level is "debug".
func (app *Application) SetControllersNoLog(disable bool) *Application {
	app.controllersNoLog = disable
	return app
}

// EnableStructDependents will try to resolve
// the fields of a struct value, if any, when it's a dependent struct value
// based on the previous registered dependencies.
func (app *Application) EnableStructDependents() *Application {
	app.container.EnableStructDependents = true
	return app
}

// Register appends one or more values as dependencies.
// The value can be a single struct value-instance or a function
// which has one input and one output, the input should be
// an `iris.Context` and the output can be any type, that output type
// will be bind-ed to the controller's field, if matching or to the
// controller's methods, if matching.
//
// These dependencies "dependencies" can be changed per-controller as well,
// via controller's `BeforeActivation` and `AfterActivation` methods,
// look the `Handle` method for more.
//
// It returns this Application.
//
// Example: `.Register(loggerService{prefix: "dev"}, func(ctx iris.Context) User {...})`.
func (app *Application) Register(dependencies ...interface{}) *Application {
	if len(dependencies) > 0 && len(app.container.Dependencies) == len(hero.BuiltinDependencies) && len(app.Controllers) > 0 {
		allControllerNamesSoFar := make([]string, len(app.Controllers))
		for i := range app.Controllers {
			allControllerNamesSoFar[i] = app.Controllers[i].Name()
		}

		golog.Warnf(`mvc.Application#Register called after mvc.Application#Handle.
	The controllers[%s] may miss required dependencies.
	Set the Logger's Level to "debug" to view the active dependencies per controller.`, strings.Join(allControllerNamesSoFar, ","))
	}

	for _, dependency := range dependencies {
		app.container.Register(dependency)
	}

	return app
}

type (
	// Option is an interface which does contain a single `Apply` method that accepts
	// a `ControllerActivator`. It can be passed on `Application.Handle` method to
	// mdoify the behavior right after the `BeforeActivation` state.
	//
	// See `GRPC` package-level structure
	// and `Version` package-level function too.
	Option interface {
		Apply(*ControllerActivator)
	}

	// OptionFunc is the functional type of `Option`.
	// Read `Option` docs.
	OptionFunc func(*ControllerActivator)
)

// Apply completes the `Option` interface.
func (opt OptionFunc) Apply(c *ControllerActivator) {
	opt(c)
}

// IgnoreEmbedded is an Option which can be used to ignore all embedded struct's method handlers.
// Note that even if the controller overrides the embedded methods
// they will be still ignored because Go doesn't support this detection so far.
// For global affect, set the `IgnoreEmbeddedControllers` package-level variable to true.
var IgnoreEmbedded OptionFunc = func(c *ControllerActivator) {
	c.SkipEmbeddedMethods()
}

// Handle serves a controller for the current mvc application's Router.
// It accept any custom struct which its functions will be transformed
// to routes.
//
// If "controller" has `BeforeActivation(b mvc.BeforeActivation)`
// or/and `AfterActivation(a mvc.AfterActivation)` then these will be called between the controller's `.activate`,
// use those when you want to modify the controller before or/and after
// the controller will be registered to the main Iris Application.
//
// It returns this mvc Application.
//
// Usage: `.Handle(new(TodoController))`.
//
// Controller accepts a sub router and registers any custom struct
// as controller, if struct doesn't have any compatible methods
// neither are registered via `ControllerActivator`'s `Handle` method
// then the controller is not registered at all.
//
// A Controller may have one or more methods
// that are wrapped to a handler and registered as routes before the server ran.
// The controller's method can accept any input argument that are previously binded
// via the dependencies or route's path accepts dynamic path parameters.
// The controller's fields are also bindable via the dependencies, either a
// static value (service) or a function (dynamically) which accepts a context
// and returns a single value (this type is being used to find the relative field or method's input argument).
//
// func(c *ExampleController) Get() string |
// (string, string) |
// (string, int) |
// int |
// (int, string |
// (string, error) |
// bool |
// (any, bool) |
// error |
// (int, error) |
// (customStruct, error) |
// customStruct |
// (customStruct, int) |
// (customStruct, string) |
// Result or (Result, error)
// where Get is an HTTP Method func.
//
// Default behavior can be changed through second, variadic, variable "options",
// e.g. Handle(controller, GRPC {Server: grpcServer, Strict: true})
//
// Examples at: https://github.com/kataras/iris/tree/main/_examples/mvc
func (app *Application) Handle(controller interface{}, options ...Option) *Application {
	c := app.handle(controller, options...)
	// Note: log on register-time, so they can catch any failures before build.
	if !app.controllersNoLog {
		// log only http (and versioned) or grpc controllers,
		// websocket is already logging itself.
		logController(app.Router.Logger(), c)
	}
	return app
}

// HandleWebsocket handles a websocket specific controller.
// Its exported methods are the events.
// If a "Namespace" field or method exists then namespace is set, otherwise empty namespace.
// Note that a websocket controller is registered and ran under a specific connection connected to a namespace
// and it cannot send HTTP responses on that state.
// However all static and dynamic dependency injection features are working, as expected, like any regular MVC Controller.
func (app *Application) HandleWebsocket(controller interface{}) *websocket.Struct {
	c := app.handle(controller)
	c.markAsWebsocket()

	websocketController := websocket.NewStruct(c.Value).SetInjector(makeInjector(c.injector))
	app.websocketControllers = append(app.websocketControllers, websocketController)
	return websocketController
}

func makeInjector(s *hero.Struct) websocket.StructInjector {
	return func(_ reflect.Type, nsConn *websocket.NSConn) reflect.Value {
		v, _ := s.Acquire(websocket.GetContext(nsConn.Conn))
		return v
	}
}

var _ websocket.ConnHandler = (*Application)(nil)

// GetNamespaces completes the websocket ConnHandler interface.
// It returns a collection of namespace and events that
// were registered through `HandleWebsocket` controllers.
func (app *Application) GetNamespaces() websocket.Namespaces {
	if logger := app.Router.Logger(); logger.Level == golog.DebugLevel && !app.controllersNoLog {
		websocket.EnableDebug(logger)
	}

	return websocket.JoinConnHandlers(app.websocketControllers...).GetNamespaces()
}

func (app *Application) handle(controller interface{}, options ...Option) *ControllerActivator {
	// initialize the controller's activator, nothing too magical so far.
	c := newControllerActivator(app, controller)

	// check the controller's "BeforeActivation" or/and "AfterActivation" method(s) between the `activate`
	// call, which is simply parses the controller's methods, end-dev can register custom controller's methods
	// by using the BeforeActivation's (a ControllerActivation) `.Handle` method.
	if before, ok := controller.(interface {
		BeforeActivation(BeforeActivation)
	}); ok {
		before.BeforeActivation(c)
	}

	for _, opt := range options {
		if opt != nil {
			opt.Apply(c)
		}
	}

	c.activate()

	if after, okAfter := controller.(interface {
		AfterActivation(AfterActivation)
	}); okAfter {
		after.AfterActivation(c)
	}

	app.Controllers = append(app.Controllers, c)
	return c
}

// HandleError registers a `hero.ErrorHandlerFunc` which will be fired when
// application's controllers' functions returns an non-nil error.
// Each controller can override it by implementing the `hero.ErrorHandler`.
func (app *Application) HandleError(handler func(ctx *context.Context, err error)) *Application {
	errorHandler := hero.ErrorHandlerFunc(handler)
	app.container.GetErrorHandler = func(*context.Context) hero.ErrorHandler {
		return errorHandler
	}
	return app
}

// Clone returns a new mvc Application which has the dependencies
// of the current mvc Application's `Dependencies` and its `ErrorHandler`.
//
// Example: `.Clone(app.Party("/path")).Handle(new(TodoSubController))`.
func (app *Application) Clone(party router.Party) *Application {
	cloned := newApp(party, app.container.Clone())
	cloned.controllersNoLog = app.controllersNoLog
	return cloned
}

// Party returns a new child mvc Application based on the current path + "relativePath".
// The new mvc Application has the same dependencies of the current mvc Application,
// until otherwise specified later manually.
//
// The router's root path of this child will be the current mvc Application's root path + "relativePath".
func (app *Application) Party(relativePath string, middleware ...context.Handler) *Application {
	return app.Clone(app.Router.Party(relativePath, middleware...))
}

var childNameReplacer = strings.NewReplacer("*", "", "(", "", ")", "")

func getArrowSymbol(static bool, field bool) string {
	if field {
		if static {
			return "╺"
		}
		return "⦿"

	}

	if static {
		return "•"
	}

	return "⦿"
}

// TODO: instead of this I want to get in touch with tools like "graphviz"
// so we can put all that information (and the API) inside web graphs,
// it will be easier for developers to see the flow of the whole application,
// but probalby I will never find time for that as we have higher priorities...just a reminder though.
func logController(logger *golog.Logger, c *ControllerActivator) {
	if logger.Level != golog.DebugLevel {
		return
	}

	if c.injector == nil { // when no actual controller methods are registered.
		return
	}

	/*
		[DBUG] controller.GreetController
		  ╺ Service         → ./service/greet_service.go:16
		  ╺ Get
		      GET /greet
			• iris.Context
			• service.Other	→ ./service/other_service.go:11
	*/

	bckpNewLine := logger.NewLine
	bckpTimeFormat := logger.TimeFormat
	logger.NewLine = false
	logger.TimeFormat = ""

	printer := logger.Printer
	reports := c.injector.Container.Reports
	ctrlName := c.RelName()
	ctrlScopeType := ""
	if !c.injector.Singleton {
		ctrlScopeType = getArrowSymbol(false, false) + " "
	}
	logger.Debugf("%s%s\n", ctrlScopeType, ctrlName)

	longestNameLen := 0
	for _, report := range reports {
		for _, entry := range report.Entries {
			if n := len(entry.InputFieldName); n > longestNameLen {
				if strings.HasSuffix(entry.InputFieldName, ctrlName) {
					continue
				}
				longestNameLen = n
			}
		}
	}

	longestMethodName := 0
	for methodName := range c.routes {
		if n := len(methodName); n > longestMethodName {
			longestMethodName = n
		}
	}

	lastColorCode := -1

	for _, report := range reports {

		childName := childNameReplacer.Replace(report.Name)
		if idx := strings.Index(childName, c.Name()); idx >= 0 {
			childName = childName[idx+len(c.Name()):] // it's always +1 otherwise should be reported as BUG.
		}

		if childName != "" && childName[0] == '.' {
			// It's a struct's method.

			childName = childName[1:]

			for _, route := range c.routes[childName] {
				if route.NoLog {
					continue
				}

				// Let them be logged again with the middlewares, e.g UseRouter or UseGlobal after this MVC app created.
				// route.NoLog = true

				colorCode := router.TraceTitleColorCode(route.Method)

				// group same methods (or errors).
				if lastColorCode == -1 {
					lastColorCode = colorCode
				} else if lastColorCode != colorCode {
					lastColorCode = colorCode
					fmt.Fprintln(printer)
				}

				fmt.Fprint(printer, "  ╺ ")
				pio.WriteRich(printer, childName, colorCode)

				entries := report.Entries[1:] // the ctrl value is always the first input argument so 1:..
				if len(entries) == 0 {
					fmt.Print("()")
				}
				fmt.Fprintln(printer)

				// pio.WriteRich(printer, "      "+route.GetTitle(), colorCode)
				fmt.Fprintf(printer, "      %s\n", route.String())

				for _, entry := range entries {
					fileLine := ""
					if !strings.Contains(entry.DependencyFile, "kataras/iris/") {
						fileLine = fmt.Sprintf("→ %s:%d", entry.DependencyFile, entry.DependencyLine)
					}

					fieldName := entry.InputFieldName

					spaceRequired := longestNameLen - len(fieldName)
					if spaceRequired < 0 {
						spaceRequired = 0
					}
					//    → ⊳ ↔
					fmt.Fprintf(printer, "    • %s%s %s\n", fieldName, strings.Repeat(" ", spaceRequired), fileLine)
				}
			}
		} else {
			// It's a struct's field.
			for _, entry := range report.Entries {
				fileLine := ""
				if !strings.Contains(entry.DependencyFile, "kataras/iris/") {
					fileLine = fmt.Sprintf("→ %s:%d", entry.DependencyFile, entry.DependencyLine)
				}

				fieldName := entry.InputFieldName
				spaceRequired := longestNameLen + 2 - len(fieldName) // plus the two spaces because it's not collapsed.
				if spaceRequired < 0 {
					spaceRequired = 0
				}

				arrowSymbol := getArrowSymbol(entry.Static, true)
				fmt.Fprintf(printer, "  %s %s%s %s\n", arrowSymbol, fieldName, strings.Repeat(" ", spaceRequired), fileLine)
			}
		}
	}
	// fmt.Fprintln(printer)

	logger.NewLine = bckpNewLine
	logger.TimeFormat = bckpTimeFormat
}
