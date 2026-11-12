package router

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/dep"
	"github.com/kataras/iris/v14/core/pkg/reflex"
)

// DI returns the Dependency Injection Container of this Router.
// Useful to configure the dependencies and inject them on the controllers and handlers.
func (router *RouterBuilder) DI() *dep.Container {
	return router.diContainer
}

// RegisterDependency registers a single or multiple dependencies to the router's DI Container.
// It's a shortcut of `router.DI().RegisterDependency`.
// It's useful when you want to register static dependencies upfront before the controllers' registration.
func (router *RouterBuilder) RegisterDependency(deps ...any) error {
	for _, dep := range deps {
		if dep == nil {
			continue
		}

		err := router.diContainer.RegisterDependency(dep)
		if err != nil {
			return err
		}
	}

	return nil
}

// HandlerFunc resolves a function which returns a Handler as a Handler.
// It accepts a function with arguments that are resolved by the registered dependencies and returns a Handler (and an optional error).
// If the returned error is not nil then it panics with that specific error.
// It panics if the given function returns nil or a different type than Handler.
//
// Example:
//
//	app.RegisterDependency(func(otherDependnecies...) *MyService {
//		return &MyService{...}
//	})
//
//	handler := app.HandlerFunc(func(service *MyService) iris.Handler {
//		return func(ctx iris.Context) {
//			ctx.WriteString(service.Get())
//		}
//	})
//
//	app.Get("/", handler)
//
// Or as middleware:
//
//	app.Use("/", app.HandlerFunc(func(service *MyService) iris.Handler {
//		return func(ctx iris.Context) {
//			// [middleware action...] before the main handler.
//			ctx.Next()
//		}
//	}))
func (router *RouterBuilder) HandlerFunc(fn any) context.Handler {
	if fn == nil {
		return nil
	}

	// Temporary dependency to get the result of the function.
	// The result must be a Handler.
	result, err := router.diContainer.InvokeReturningFirst(fn)
	if err != nil {
		panic(fmt.Errorf("handlerfunc: invoke: %w", err))
	}

	if result == nil {
		panic(fmt.Errorf("handlerfunc: the given function returned nil: %#+v", fn))
	}

	// Try to convert the result to a Handler if it's not already.
	// So something like Handler(dependency *Service) func(w http.ResponseWriter, r *http.Request)
	// could work as well.
	if handler, err := context.TryFromStd(result); err == nil {
		return handler
	}

	// The result must be a Handler.
	handler, ok := result.(context.Handler)
	if !ok {
		panic(fmt.Errorf("handlerfunc: the given function must return a Handler but it results to a different type of: %T", result))
	}
	return handler
}

// HandlerFuncs resolves a list of functions which return functions with dependencies as Handlers.
// Read the `HandlerFunc` documentation for more information.
func (router *RouterBuilder) HandlerFuncs(fns ...any) context.Handlers {
	handlers := make(context.Handlers, 0, len(fns))
	for _, fn := range fns {
		if fn == nil {
			continue
		}

		handlers = append(handlers, router.HandlerFunc(fn))
	}

	return handlers
}

type (
	// Controller is an interface which all child routers that are registered
	// through `Controller` should implement.
	Controller interface {
		// Routing should register the routes of the controller.
		Routing(child Router)
	}

	// Routing is a type which can be used to convert a function to a Controller.
	Routing func(Router)

	routingReturnsController interface {
		Routing() Controller
	}

	// controllerFlowManager is an optional interface which a `Controller`
	// can implement to execute code after the controller's registration.
	controllerFlowManager interface {
		// AfterRegister is called after the controller's registration.
		AfterRegister(r Router, c any, isLast bool)
	}
)

// Routing implements the Controller interface.
func (f Routing) Routing(r Router) {
	f(r)
}

// ControllerResolver turns an arbitrary value into a Controller. It reports
// false when the value is none of its business, so the next resolver gets a
// turn. See RegisterControllerResolver.
type ControllerResolver func(any) (Controller, bool)

var controllerResolvers = []ControllerResolver{}

var builtinControllerResolvers = []ControllerResolver{ // these builtin resolvers can exist side by side with the custom ones, and they are executed first.
	controllerStructMethod,
}

// RegisterControllerResolver registers a new ControllerResolver.
// It's useful when you want to register a custom controller resolver.
// The last registered resolver has the highest priority.
// Use it before of the `Router.Router` method call.
//
// It can register any method that can resolve a Controller directly or even
// after its dependencies are resolved from a function outputs the controller.
func RegisterControllerResolver(resolver ControllerResolver) {
	controllerResolvers = append(controllerResolvers, resolver)
}

func resolveController(c any) (Controller, bool) {
	var (
		builtinController Controller
		builtinOK         bool
		ctrl              Controller
		ok                bool
	)
	for _, resolver := range slices.Backward(builtinControllerResolvers) {
		if builtinController, builtinOK = resolver(c); builtinOK { // stop on first resolved.
			break
		}
	}

	for _, resolver := range slices.Backward(controllerResolvers) {
		if ctrl, ok = resolver(c); ok { // stop on first resolved.
			break
		}
	}

	return Join(builtinController, ctrl), builtinOK || ok
}

// Router creates and returns a new child Router based on a prefix path.
// It accepts a relative path and one or more handlers or controllers and returns the child group.
//
// Simple group usage:
// It accepts one or more handlers as middlewares.
// If the function is a handler, then it's registered as a middleware to the child router only for matched routes.
//
//	usersAPI := app.Router("/users", authMiddleware, logMiddleware)
//	usersAPI.Handle("GET", ...)
//
// Router func usage:
// It accepts a function that receives this created child Router.
//
//	app.Router("/users", func(r iris.Router) {
//		r.Use(authMiddleware, logMiddleware)
//		r.Get("/", getAllUsers)
//		r.Post("/", createOrUpdateUser)
//		r.Delete("/", deleteUser)
//	})
//
// Routing interface usage:
// It accepts a pointer to a struct value which implements the Routing interface: Routing(r Router) method.
// Useful when the api's dependencies amount are too much to pass on a function.
//
//	app.Router("/users", &api.UserController{UserRepository: ..., ...})
//
// Where UserController looks like:
//
//	type UserController struct { Users UserRepository }
//	func(c *UserController) Routing(r iris.Router) {
//	 r.Get("/{id:uuid}", c.getUser)
//	 [...]
//	}
//
// Usage with (static) dependencies:
//
//	app.RegisterDependency(userRepo, ...)
//	app.Router("/users", new(router.UserController))
//
// Controller usage:
// A controller can be any type of the following:
//
//   - A struct which implements the Controller interface.
//   - A ptr to struct which implements the Controller interface and its exported fields are binded based on the registered dependencies.
//   - A function which returns a struct which implements the Controller interface.
//   - A function which accepts a Router and returns nothing.
//   - A function which accepts a Router and returns a struct which implements the Controller interface.
//   - A function which returns a struct which implements the Controller interface and accepts a Router.
//   - A function which returns a struct which implements the Controller interface and accepts a Router and
//     a pointer to a struct which its fields are binded based on the registered dependencies.
func (router *RouterBuilder) Router(relativePath string, controllers ...any) Router {
	var child Router

	if relativePath == "" || relativePath == "/" || relativePath == "." {
		child = router
	} else {
		child = router.child(relativePath)
	}

	for i, c := range controllers {
		if c == nil {
			continue
		}

		if childFunc, ok := c.(func(Router)); ok { // a plain function that receives the child Router.
			childFunc(child)
			continue // it is one controller of many, keep registering the rest.
		}

		// If it's a Handler then use it directly as a middleware.
		if irisHandler, ok := c.(context.Handler); ok {
			child.Use(irisHandler)
			continue
		}

		if irisHandlers, ok := c.(context.Handlers); ok {
			child.Use(irisHandlers...)
			continue
		}

		// Check if it's a controller.
		controller, ok := c.(Controller)
		if ok && reflex.IsStructPtr(c) {
			// If it's just a pointer to a struct then bind its fields based on the registered dependencies.
			if err := router.diContainer.BindStructFields(reflex.ValueOf(c)); err != nil {
				panic(err)
			}
		} else if !ok {
			// Check if it's a function which returns a controller.
			if routingCtrl, isRoutingCtrl := c.(routingReturnsController); isRoutingCtrl {
				controller = routingCtrl.Routing()
				ok = true // proceed as a controller.
			}
		}
		if ok {
			// Bind the exported fields by the function input arguments if it's already a Controller (structure).
			if reflex.IsStructPtr(controller) {
				_, err := router.diContainer.Invoke(controller)
				if err != nil {
					panic(err)
				}
			}
		} else {
			// try to resolve it as a controller by giving flexibility to the end-developer.
			controller, ok = resolveController(c)
			if !ok {
				// else try to resolve it from a function.
				controller = spawnController(c)
			}
		}

		controller.Routing(child)

		if flow, ok := c.(controllerFlowManager); ok {
			// execute this only after all controllers of the same call were registered,
			// so dependencies apply to all of them.
			isLast := len(controllers)-1 == i
			flow.AfterRegister(child, c, isLast)
		}
	}

	return child
}

// Router's Controller helpers.

func spawnController(controllerConstructor any) Controller {
	if controllerConstructor == nil {
		panic("iris: router: Router: a nil controller cannot be registered")
	}

	return Routing(func(r Router) {
		// Temporary dependency to get the result of the function.
		result, err := r.DI().InvokeReturningFirst(controllerConstructor)
		if err != nil {
			panic(fmt.Errorf("iris: router: Router: %s: %T: %w", r.GetRelPath(), controllerConstructor, err))
		}
		if result == nil {
			panic(fmt.Sprintf("iris: router: Router: %s: %T returned nil; return a controller value, not a nil one", r.GetRelPath(), controllerConstructor))
		}

		// Cast result as Controller, if the given function doesn't return a Controller try as func(router.Router), and if not then panic.
		var c Controller
		switch val := result.(type) {
		case Controller:
			c = val
		case routingReturnsController:
			c = val.Routing()
		case func(Router): // a plain function that receives the child Router.
			c = Routing(val)
		default:
			// try to resolve it as a controller by giving flexibility to the end-developer.
			if structCtrl, ok := resolveController(result); ok {
				c = structCtrl
			} else {
				panic(fmt.Sprintf("iris: router: Router: %s: %T returned a %T; return a value with a Routing(iris.Router) method, or a func(iris.Router)", r.GetRelPath(), controllerConstructor, result))
			}
		}

		// call its underline routing.
		c.Routing(r)
	})
	// We could use the `Services` helper to register this dependency only for this but we still have to inspect the dependencies to get the actual controller value, so we don't.
}

// ResolveController returns a Controller which was already registered as a dependency to the Router before.
func ResolveController[T any]() Controller {
	// Use the pointer of it always, the T must implement the Controller interface on its pointer value as documented. Don't call newController
	// because it will not get the already-resolved controller but it will create new and it will only try to bind its fields based on dependencies instead of using the existing-created one on dependencies list.
	// return spawnController(&t)

	return Routing(func(r Router) {
		resolved, ok := r.DI().Resolve[*T]()
		if !ok {
			panic(fmt.Sprintf("iris: router: ResolveController: %s: *%s is not registered; register it with RegisterDependency (or the builder's Services) before calling ResolveController", r.GetRelPath(), reflect.TypeFor[T]()))
		}

		ctrl, ok := any(resolved).(Controller)
		if !ok {
			panic(fmt.Sprintf("iris: router: ResolveController: %s: the resolved %T does not implement the Controller interface; give it a Routing(iris.Router) method", r.GetRelPath(), resolved))
		}

		ctrl.Routing(r)
	})
}

// Join is a Controller helper which can be used as a shortcut to join multiple controllers into one.
// Useful when you want to group multiple controllers under the same path and the call expects a single Controller output parameter.
func Join(controllers ...any) Controller {
	return Routing(func(r Router) {
		r.Router("/", controllers...)

		/* and if we ever need to skip the builtin support for dependency injection inside these type of controllers too:
		for _, ctrl := range controllers {
			ctrl.Routing(r)
		}*/
	})
}

// Prefix is a Controller helper which can be used
// as a shortcut to add a relative path to all controllers passed to it.
func Prefix(relativePath string, controllers ...any) Controller {
	return Routing(func(r Router) {
		r.Router(relativePath, controllers...)
	})
}

// Middleware is a Controller helper which can be used
// as a shortcut to add middlewares on Router.Router("/path", Middleware(handler), new(example.API)).
func Middleware(handlers ...context.Handler) Controller {
	return Routing(func(r Router) {
		r.Use(handlers...)
	})
}

type controllerDependencies struct {
	deps []any

	routerContainer *dep.Container
	rb              *RouterBuilder
}

var _ controllerFlowManager = (*controllerDependencies)(nil)

// Dependencies returns a Controller which can be used to register
// dependencies to the controllers follows in the same `Router.Router` method call.
// By default controllers will inherit the dependencies registered to the Router,
// this method overrides the inherited dependencies for the controllers only.
//
// Useful when you have the same interface declared for different services
// that can be binded to more than one controller.
func Dependencies(deps ...any) Controller {
	return &controllerDependencies{deps: deps}
}

func (p *controllerDependencies) Routing(r Router) {
	rb, ok := r.(*RouterBuilder)
	if !ok {
		// Unreachable through Router.Router, which always passes its own
		// *RouterBuilder. A hand-written Router implementation would land here.
		panic(fmt.Sprintf("iris: router: Dependencies: %s: needs the builtin Router implementation, got %T", r.GetRelPath(), r))
	}
	p.rb = rb

	p.registerDependencies()
}

func (p *controllerDependencies) registerDependencies() {
	p.routerContainer = p.rb.diContainer

	controllerContainer := p.rb.diContainer.Clone()
	controllerContainer.RegisterDependency(p.deps...)

	// replace the whole container with the new one until it's resetted on AfterRegister method.
	p.rb.diContainer = controllerContainer
}

func (p *controllerDependencies) AfterRegister(r Router, c any, isLast bool) {
	if isLast {
		p.rb.diContainer = p.routerContainer // restore the router's container.
	}
}

type (
	ctrlGetMethod interface {
		Get(*context.Context)
	}

	ctrlPostMethod interface {
		Post(*context.Context)
	}

	ctrlPutMethod interface {
		Put(*context.Context)
	}

	ctrlDeleteMethod interface {
		Delete(*context.Context)
	}

	ctrlPatchMethod interface {
		Patch(*context.Context)
	}

	ctrlOptionsMethod interface {
		Options(*context.Context)
	}

	ctrlHeadMethod interface {
		Head(*context.Context)
	}

	ctrlConnectMethod interface {
		Connect(*context.Context)
	}

	ctrlTraceMethod interface {
		Trace(*context.Context)
	}

	ctrlQueryMethod interface {
		Query(*context.Context)
	}
)

func controllerStructMethod(ctrl any) (Controller, bool) {
	if ctrl == nil {
		return nil, false
	}

	getMethod, hasGetMethod := ctrl.(ctrlGetMethod)
	postMethod, hasPostMethod := ctrl.(ctrlPostMethod)
	putMethod, hasPutMethod := ctrl.(ctrlPutMethod)
	deleteMethod, hasDeleteMethod := ctrl.(ctrlDeleteMethod)
	patchMethod, hasPatchMethod := ctrl.(ctrlPatchMethod)
	optionsMethod, hasOptionsMethod := ctrl.(ctrlOptionsMethod)
	headMethod, hasHeadMethod := ctrl.(ctrlHeadMethod)
	connectMethod, hasConnectMethod := ctrl.(ctrlConnectMethod)
	traceMethod, hasTraceMethod := ctrl.(ctrlTraceMethod)
	queryMethod, hasQueryMethod := ctrl.(ctrlQueryMethod)

	if !hasGetMethod && !hasPostMethod && !hasPutMethod && !hasDeleteMethod && !hasPatchMethod && !hasOptionsMethod && !hasHeadMethod && !hasConnectMethod && !hasTraceMethod && !hasQueryMethod {
		return nil, false
	}

	c, isControllerToo := ctrl.(Controller) // check for this afterwards, because there is already check for the Controller implementation.
	typ := reflect.TypeOf(ctrl)

	controller := Routing(func(r Router) {
		// Note(@kataras): Routing runs BEFORE the verb routes are registered.
		//
		// A route captures the router's middleware when it is created, so an
		// r.Use inside Routing cannot reach a route that already exists. Running
		// it last left every Get/Post/Put/Delete/... method below unguarded while
		// the developer believed the whole controller was protected.
		if isControllerToo {
			c.Routing(r)
		}

		if hasGetMethod {
			route := r.Get("/", getMethod.Get)
			route.SetSourceByMethodType(typ, "Get")
			// } else if hasGetErrMethod {
			// r.Get("/", errors.Handler(getErrMethod.Get))
			// NOTE:
			// I don't like importing the rest package here, but it's the best way to keep the code clean.
			// The other way is to export a function which registers controllers resolvers and force the end-developer to import _ rest package in order to be resolvable.
			// Actually this is not a bad idea at all, it will make the framework even more modular and the end-developer so much more flexible. OK done with `RegisterControllerResolver` and `errors/controller.go` file.
		}

		if hasPostMethod {
			route := r.Post("/", postMethod.Post)
			route.SetSourceByMethodType(typ, "Post")
		}
		if hasPutMethod {
			route := r.Put("/", putMethod.Put)
			route.SetSourceByMethodType(typ, "Put")
		}
		if hasDeleteMethod {
			route := r.Delete("/", deleteMethod.Delete)
			route.SetSourceByMethodType(typ, "Delete")
		}
		if hasPatchMethod {
			route := r.Patch("/", patchMethod.Patch)
			route.SetSourceByMethodType(typ, "Patch")
		}
		if hasOptionsMethod {
			route := r.Options("/", optionsMethod.Options)
			route.SetSourceByMethodType(typ, "Options")
		}
		if hasHeadMethod {
			route := r.Head("/", headMethod.Head)
			route.SetSourceByMethodType(typ, "Head")
		}
		if hasConnectMethod {
			route := r.Connect("/", connectMethod.Connect)
			route.SetSourceByMethodType(typ, "Connect")
		}
		if hasTraceMethod {
			route := r.Trace("/", traceMethod.Trace)
			route.SetSourceByMethodType(typ, "Trace")
		}
		if hasQueryMethod {
			route := r.Query("/", queryMethod.Query)
			route.SetSourceByMethodType(typ, "Query")
		}
	})

	return controller, true
}
