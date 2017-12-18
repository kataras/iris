package mvc

import "github.com/kataras/iris/core/router"

// Application is the high-level compoment of the "mvc" package.
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
	Engine *Engine
	Router router.Party
}

func newApp(engine *Engine, subRouter router.Party) *Application {
	return &Application{
		Engine: engine,
		Router: subRouter,
	}
}

// New returns a new mvc Application based on a "party".
// Application creates a new engine which is responsible for binding the dependencies
// and creating and activating the app's controller(s).
//
// Example: `New(app.Party("/todo"))` or `New(app)` as it's the same as `New(app.Party("/"))`.
func New(party router.Party) *Application {
	return newApp(NewEngine(), party)
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

// AddDependencies adds one or more values as dependencies.
// The value can be a single struct value-instance or a function
// which has one input and one output, the input should be
// an `iris.Context` and the output can be any type, that output type
// will be binded to the controller's field, if matching or to the
// controller's methods, if matching.
//
// The dependencies can be changed per-controller as well via a `beforeActivate`
// on the `Register` method or when the controller has the `BeforeActivation(b BeforeActivation)`
// method defined.
//
// It returns this Application.
//
// Example: `.AddDependencies(loggerService{prefix: "dev"}, func(ctx iris.Context) User {...})`.
func (app *Application) AddDependencies(values ...interface{}) *Application {
	app.Engine.Dependencies.Add(values...)
	return app
}

// Register adds a controller for the current Router.
// It accept any custom struct which its functions will be transformed
// to routes.
//
// The second, optional and variadic argument is the "beforeActive",
// use that when you want to modify the controller before the activation
// and registration to the main Iris Application.
//
// It returns this Application.
//
// Example: `.Register(new(TodoController))`.
func (app *Application) Register(controller interface{}, beforeActivate ...func(BeforeActivation)) *Application {
	app.Engine.Controller(app.Router, controller, beforeActivate...)
	return app
}

// NewChild creates and returns a new MVC Application which will be adapted
// to the "party", it adopts
// the parent's (current) dependencies, the "party" may be
// a totally new router or a child path one via the parent's `.Router.Party`.
//
// Example: `.NewChild(irisApp.Party("/path")).Register(new(TodoSubController))`.
func (app *Application) NewChild(party router.Party) *Application {
	return newApp(app.Engine.Clone(), party)
}
