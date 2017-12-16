package mvc

import (
	"github.com/kataras/golog"
	"github.com/kataras/iris/mvc/di"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
)

// Engine contains the Dependencies which will be binded
// to the controller(s) or handler(s) that can be created
// using the Engine's `Handler` and `Controller` methods.
//
// This is not exported for being used by everyone, use it only when you want
// to share engines between multi mvc.go#Application
// or make custom mvc handlers that can be used on the standard
// iris' APIBuilder. The last one reason is the most useful here,
// although end-devs can use the `MakeHandler` as well.
//
// For a more high-level structure please take a look at the "mvc.go#Application".
type Engine struct {
	Dependencies *di.D
}

// NewEngine returns a new engine, a container for dependencies and a factory
// for handlers and controllers, this is used internally by the `mvc#Application` structure.
// Please take a look at the structure's documentation for more information.
func NewEngine() *Engine {
	return &Engine{
		Dependencies: di.New().Hijack(hijacker).GoodFunc(typeChecker),
	}
}

// Clone creates and returns a new engine with the parent's(current) Dependencies.
// It copies the current "e" dependencies and returns a new engine.
func (e *Engine) Clone() *Engine {
	child := NewEngine()
	child.Dependencies = e.Dependencies.Clone()
	return child
}

// Handler accepts a "handler" function which can accept any input arguments that match
// with the Engine's `Dependencies` and any output result; like string, int (string,int),
// custom structs, Result(View | Response) and anything you already know that mvc implementation supports.
// It returns a standard `iris/context.Handler` which can be used anywhere in an Iris Application,
// as middleware or as simple route handler or subdomain's handler.
func (e *Engine) Handler(handler interface{}) context.Handler {
	h, err := MakeHandler(handler, e.Dependencies.Values...)
	if err != nil {
		golog.Errorf("mvc handler: %v", err)
	}
	return h
}

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
// Examples at: https://github.com/kataras/iris/tree/master/_examples/mvc.
func (e *Engine) Controller(router router.Party, controller interface{}, beforeActivate ...func(*ControllerActivator)) {
	ca := newControllerActivator(router, controller, e.Dependencies)

	// give a priority to the "beforeActivate"
	// callbacks, if any.
	for _, cb := range beforeActivate {
		cb(ca)
	}

	// check if controller has an "BeforeActivate" function
	// which accepts the controller activator and call it.
	if activateListener, ok := controller.(interface {
		BeforeActivate(*ControllerActivator)
	}); ok {
		activateListener.BeforeActivate(ca)
	}

	ca.activate()
}
