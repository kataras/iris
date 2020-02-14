package mvc

import (
	"reflect"
	"strings"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/hero"
	"github.com/kataras/iris/v12/hero/di"
	"github.com/kataras/iris/v12/websocket"

	"github.com/kataras/golog"
)

// HeroDependencies let you share bindable dependencies between
// package-level hero's registered dependencies and all MVC instances that comes later.
//
// `hero.Register(...)`
// `myMVC := mvc.New(app.Party(...))`
// the "myMVC" registers the dependencies provided by the `hero.Register` func
// automatically.
//
// Set it to false to disable that behavior, you have to use the `mvc#Register`
// even if you had register dependencies with the `hero` package.
//
// Defaults to true.
var HeroDependencies = true

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
	Dependencies di.Values
	// Sorter is a `di.Sorter`, can be used to customize the order of controller's fields
	// and their available bindable values to set.
	// Sorting matters only when a field can accept more than one registered value.
	// Defaults to nil; order of registration matters when more than one field can accept the same value.
	Sorter               di.Sorter
	Router               router.Party
	Controllers          []*ControllerActivator
	websocketControllers []websocket.ConnHandler
	ErrorHandler         di.ErrorHandler
}

func newApp(subRouter router.Party, values di.Values) *Application {
	return &Application{
		Router:       subRouter,
		Dependencies: values,
		ErrorHandler: di.DefaultErrorHandler,
	}
}

// New returns a new mvc Application based on a "party".
// Application creates a new engine which is responsible for binding the dependencies
// and creating and activating the app's controller(s).
//
// Example: `New(app.Party("/todo"))` or `New(app)` as it's the same as `New(app.Party("/"))`.
func New(party router.Party) *Application {
	values := di.NewValues()
	if HeroDependencies {
		values = hero.Dependencies().Clone()
	}

	return newApp(party, values)
}

// Configure creates a new controller and configures it,
// this function simply calls the `New(party)` and its `.Configure(configurators...)`.
//
// A call of `mvc.New(app.Party("/path").Configure(buildMyMVC)` is equal to
//           	 `mvc.Configure(app.Party("/path"), buildMyMVC)`.
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

// Register appends one or more values as dependencies.
// The value can be a single struct value-instance or a function
// which has one input and one output, the input should be
// an `iris.Context` and the output can be any type, that output type
// will be bind-ed to the controller's field, if matching or to the
// controller's methods, if matching.
//
// These dependencies "values" can be changed per-controller as well,
// via controller's `BeforeActivation` and `AfterActivation` methods,
// look the `Handle` method for more.
//
// It returns this Application.
//
// Example: `.Register(loggerService{prefix: "dev"}, func(ctx iris.Context) User {...})`.
func (app *Application) Register(values ...interface{}) *Application {
	if len(values) > 0 && app.Dependencies.Len() == 0 && len(app.Controllers) > 0 {
		allControllerNamesSoFar := make([]string, len(app.Controllers))
		for i := range app.Controllers {
			allControllerNamesSoFar[i] = app.Controllers[i].Name()
		}

		golog.Warnf(`mvc.Application#Register called after mvc.Application#Handle.
The controllers[%s] may miss required dependencies.
Set the Logger's Level to "debug" to view the active dependencies per controller.`, strings.Join(allControllerNamesSoFar, ","))
	}

	app.Dependencies.Add(values...)

	return app
}

// SortByNumMethods is the same as `app.Sorter = di.SortByNumMethods` which
// prioritize fields and their available values (only if more than one)
// with the highest number of methods,
// this way an empty interface{} is getting the "thinnest" available value.
func (app *Application) SortByNumMethods() *Application {
	app.Sorter = di.SortByNumMethods
	return app
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
// Examples at: https://github.com/kataras/iris/tree/master/_examples/mvc
func (app *Application) Handle(controller interface{}) *Application {
	app.handle(controller)
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

func makeInjector(injector *di.StructInjector) websocket.StructInjector {
	return func(_ reflect.Type, nsConn *websocket.NSConn) reflect.Value {
		v := injector.Acquire()
		if injector.CanInject {
			injector.InjectElem(websocket.GetContext(nsConn.Conn), v.Elem())
		}
		return v
	}
}

var _ websocket.ConnHandler = (*Application)(nil)

// GetNamespaces completes the websocket ConnHandler interface.
// It returns a collection of namespace and events that
// were registered through `HandleWebsocket` controllers.
func (app *Application) GetNamespaces() websocket.Namespaces {
	if golog.Default.Level == golog.DebugLevel {
		websocket.EnableDebug(golog.Default)
	}

	return websocket.JoinConnHandlers(app.websocketControllers...).GetNamespaces()
}

func (app *Application) handle(controller interface{}) *ControllerActivator {
	// initialize the controller's activator, nothing too magical so far.
	c := newControllerActivator(app.Router, controller, app.Dependencies, app.Sorter, app.ErrorHandler)

	// check the controller's "BeforeActivation" or/and "AfterActivation" method(s) between the `activate`
	// call, which is simply parses the controller's methods, end-dev can register custom controller's methods
	// by using the BeforeActivation's (a ControllerActivation) `.Handle` method.
	if before, ok := controller.(interface {
		BeforeActivation(BeforeActivation)
	}); ok {
		before.BeforeActivation(c)
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
func (app *Application) HandleError(errorHandler func(ctx context.Context, err error)) *Application {
	app.ErrorHandler = di.ErrorHandlerFunc(errorHandler)
	return app
}

// Clone returns a new mvc Application which has the dependencies
// of the current mvc Application's `Dependencies` and its `ErrorHandler`.
//
// Example: `.Clone(app.Party("/path")).Handle(new(TodoSubController))`.
func (app *Application) Clone(party router.Party) *Application {
	cloned := newApp(party, app.Dependencies.Clone())
	cloned.ErrorHandler = app.ErrorHandler
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
