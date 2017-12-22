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

// AddDependencies adds one or more values as dependencies.
// The value can be a single struct value-instance or a function
// which has one input and one output, the input should be
// an `iris.Context` and the output can be any type, that output type
// will be binded to the controller's field, if matching or to the
// controller's methods, if matching.
//
// These dependencies "values" can be changed per-controller as well,
// via controller's `BeforeActivation` and `AfterActivation` methods,
// look the `Register` method for more.
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
// If "controller" has `BeforeActivation(b mvc.BeforeActivation)`
// or/and `AfterActivation(a mvc.AfterActivation)` then these will be called between the controller's `.activate`,
// use those when you want to modify the controller before or/and after
// the controller will be registered to the main Iris Application.
//
// It returns this mvc Application.
//
// Example: `.Register(new(TodoController))`.
func (app *Application) Register(controller interface{}) *Application {
	app.Engine.Controller(app.Router, controller)
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
