package mvc

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/hero"
	"github.com/kataras/iris/v12/hero/di"
	"github.com/kataras/iris/v12/macro"

	"github.com/kataras/golog"
)

// BaseController is the optional controller interface, if it's
// completed by the end controller then the BeginRequest and EndRequest
// are called between the controller's method responsible for the incoming request.
type BaseController interface {
	BeginRequest(context.Context)
	EndRequest(context.Context)
}

type shared interface {
	Name() string
	Router() router.Party
	GetRoute(methodName string) *router.Route
	GetRoutes(methodName string) []*router.Route
	Handle(httpMethod, path, funcName string, middleware ...context.Handler) *router.Route
	HandleMany(httpMethod, path, funcName string, middleware ...context.Handler) []*router.Route
}

// BeforeActivation is being used as the only one input argument of a
// `func(c *Controller) BeforeActivation(b mvc.BeforeActivation) {}`.
//
// It's being called before the controller's dependencies binding to the fields or the input arguments
// but before server ran.
//
// It's being used to customize a controller if needed inside the controller itself,
// it's called once per application.
type BeforeActivation interface {
	shared
	Dependencies() *di.Values
}

// AfterActivation is being used as the only one input argument of a
// `func(c *Controller) AfterActivation(a mvc.AfterActivation) {}`.
//
// It's being called after the `BeforeActivation`,
// and after controller's dependencies bind-ed to the fields or the input arguments but before server ran.
//
// It's being used to customize a controller if needed inside the controller itself,
// it's called once per application.
type AfterActivation interface {
	shared
	DependenciesReadOnly() ValuesReadOnly
	Singleton() bool
}

var (
	_ BeforeActivation = (*ControllerActivator)(nil)
	_ AfterActivation  = (*ControllerActivator)(nil)
)

// ControllerActivator returns a new controller type info description.
// Its functionality can be overridden by the end-dev.
type ControllerActivator struct {
	// the router is used on the `Activate` and can be used by end-dev on the `BeforeActivation`
	// to register any custom controller's methods as handlers.
	router router.Party

	macros              macro.Macros
	tmplParamStartIndex int

	// initRef BaseController // the BaseController as it's passed from the end-dev.
	Value reflect.Value // the BaseController's Value.
	Type  reflect.Type  // raw type of the BaseController (initRef).
	// FullName it's the last package path segment + "." + the Name.
	// i.e: if login-example/user/controller.go, the FullName is "user.Controller".
	fullName string

	// the already-registered routes, key = the controller's function name.
	// End-devs can change some properties of the *Route on the `BeforeActivator` by using the
	// `GetRoute/GetRoutes(functionName)`.
	routes map[string][]*router.Route

	// the bindings that comes from the Engine and the controller's filled fields if any.
	// Can be bind-ed to the the new controller's fields and method that is fired
	// on incoming requests.
	dependencies di.Values
	sorter       di.Sorter

	errorHandler di.ErrorHandler

	// initialized on the first `Handle` or immediately when "servesWebsocket" is true.
	injector *di.StructInjector

	// true if this controller listens and serves to websocket events.
	servesWebsocket bool
}

// NameOf returns the package name + the struct type's name,
// it's used to take the full name of an Controller, the `ControllerActivator#Name`.
func NameOf(v interface{}) string {
	elemTyp := di.IndirectType(di.ValueOf(v).Type())

	typName := elemTyp.Name()
	pkgPath := elemTyp.PkgPath()
	fullname := pkgPath[strings.LastIndexByte(pkgPath, '/')+1:] + "." + typName

	return fullname
}

func newControllerActivator(router router.Party, controller interface{}, dependencies []reflect.Value, sorter di.Sorter, errorHandler di.ErrorHandler) *ControllerActivator {
	typ := reflect.TypeOf(controller)

	c := &ControllerActivator{
		// give access to the Router to the end-devs if they need it for some reason,
		// i.e register done handlers.
		router: router,
		macros: *router.Macros(),
		Value:  reflect.ValueOf(controller),
		Type:   typ,
		// the full name of the controller: its type including the package path.
		fullName: NameOf(controller),
		// set some methods that end-dev cann't use accidentally
		// to register a route via the `Handle`,
		// all available exported and compatible methods
		// are being appended to the slice at the `parseMethods`,
		// if a new method is registered via `Handle` its function name
		// is also appended to that slice.
		routes: whatReservedMethods(typ),
		// CloneWithFieldsOf: include the manual fill-ed controller struct's fields to the dependencies.
		dependencies: di.Values(dependencies).CloneWithFieldsOf(controller),
		sorter:       sorter,
		errorHandler: errorHandler,
	}

	fpath, _ := macro.Parse(c.router.GetRelPath(), c.macros)
	c.tmplParamStartIndex = len(fpath.Params)
	return c
}

func whatReservedMethods(typ reflect.Type) map[string][]*router.Route {
	methods := []string{"BeforeActivation", "AfterActivation"}
	//  BeforeActivatior/AfterActivation are not routes but they are
	// reserved names*
	if isBaseController(typ) {
		methods = append(methods, "BeginRequest", "EndRequest")
	}

	routes := make(map[string][]*router.Route, len(methods))
	for _, m := range methods {
		routes[m] = []*router.Route{}
	}

	return routes
}

func (c *ControllerActivator) markAsWebsocket() {
	c.servesWebsocket = true
	c.attachInjector()
}

// Dependencies returns the write and read access of the dependencies that are
// came from the parent MVC Application, with this you can customize
// the dependencies per controller, used at the `BeforeActivation`.
func (c *ControllerActivator) Dependencies() *di.Values {
	return &c.dependencies
}

// ValuesReadOnly returns the read-only access type of the controller's dependencies.
// Used at `AfterActivation`.
type ValuesReadOnly interface {
	// Has returns true if a binder responsible to
	// bind and return a type of "typ" is already registered to this controller.
	Has(value interface{}) bool
	// Len returns the length of the values.
	Len() int
	// Clone returns a copy of the current values.
	Clone() di.Values
	// CloneWithFieldsOf will return a copy of the current values
	// plus the "s" struct's fields that are filled(non-zero) by the caller.
	CloneWithFieldsOf(s interface{}) di.Values
}

// DependenciesReadOnly returns the read-only access type of the controller's dependencies.
// Used at `AfterActivation`.
func (c *ControllerActivator) DependenciesReadOnly() ValuesReadOnly {
	return c.dependencies
}

// Name returns the full name of the controller, its package name + the type name.
// Can used at both `BeforeActivation` and `AfterActivation`.
func (c *ControllerActivator) Name() string {
	return c.fullName
}

// Router is the standard Iris router's public API.
// With this you can register middleware, view layouts, subdomains, serve static files
// and even add custom standard iris handlers as normally.
//
// This Router is the router instance that came from the parent MVC Application,
// it's the `app.Party(...)` argument.
//
// Can used at both `BeforeActivation` and `AfterActivation`.
func (c *ControllerActivator) Router() router.Party {
	return c.router
}

// GetRoute returns the first registered route based on the controller's method name.
// It can be used to change the route's name, which is useful for reverse routing
// inside views. Custom routes can be registered with `Handle`, which returns the *Route.
// This method exists mostly for the automatic method parsing based on the known patterns
// inside a controller.
//
// A check for `nil` is necessary for unregistered methods.
//
// See `GetRoutes` and `Handle` too.
func (c *ControllerActivator) GetRoute(methodName string) *router.Route {
	routes := c.GetRoutes(methodName)
	if len(routes) > 0 {
		return routes[0]
	}

	return nil
}

// GetRoutes returns one or more registered route based on the controller's method name.
// It can be used to change the route's name, which is useful for reverse routing
// inside views. Custom routes can be registered with `Handle`, which returns the *Route.
// This method exists mostly for the automatic method parsing based on the known patterns
// inside a controller.
//
// A check for `nil` is necessary for unregistered methods.
//
// See `Handle` too.
func (c *ControllerActivator) GetRoutes(methodName string) []*router.Route {
	for name, routes := range c.routes {
		if name == methodName {
			return routes
		}
	}
	return nil
}

// Singleton returns new if all incoming clients' requests
// have the same controller instance.
// This is done automatically by iris to reduce the creation
// of a new controller on each request, if the controller doesn't contain
// any unexported fields and all fields are services-like, static.
func (c *ControllerActivator) Singleton() bool {
	if c.injector == nil {
		panic("MVC: Singleton used on an invalid state the API gives access to it only `AfterActivation`, report this as bug")
	}
	return c.injector.Scope == di.Singleton
}

// checks if a method is already registered.
func (c *ControllerActivator) isReservedMethod(name string) bool {
	for methodName := range c.routes {
		if methodName == name {
			return true
		}
	}

	return false
}

func (c *ControllerActivator) activate() {
	c.parseMethods()
}

func (c *ControllerActivator) addErr(err error) bool {
	return c.router.GetReporter().Err(err) != nil
}

// register all available, exported methods to handlers if possible.
func (c *ControllerActivator) parseMethods() {
	n := c.Type.NumMethod()
	for i := 0; i < n; i++ {
		m := c.Type.Method(i)
		c.parseMethod(m)
	}
}

func (c *ControllerActivator) parseMethod(m reflect.Method) {
	httpMethod, httpPath, err := parseMethod(*c.router.Macros(), m, c.isReservedMethod)
	if err != nil {
		if err != errSkip {
			c.addErr(fmt.Errorf("MVC: fail to parse the route path and HTTP method for '%s.%s': %v", c.fullName, m.Name, err))
		}

		return
	}

	c.Handle(httpMethod, httpPath, m.Name)
}

// Handle registers a route based on a http method, the route's path
// and a function name that belongs to the controller, it accepts
// a forth, optionally, variadic parameter which is the before handlers.
//
// Just like `Party#Handle`, it returns the `*router.Route`, if failed
// then it logs the errors and it returns nil, you can check the errors
// programmatically by the `Party#GetReporter`.
func (c *ControllerActivator) Handle(method, path, funcName string, middleware ...context.Handler) *router.Route {
	routes := c.HandleMany(method, path, funcName, middleware...)
	if len(routes) == 0 {
		return nil
	}

	return routes[0]
}

// HandleMany like `Handle` but can register more than one path and HTTP method routes
// separated by whitespace on the same controller's method.
// Keep note that if the controller's method input arguments are path parameters dependencies
// they should match with each of the given paths.
//
// Just like `Party#HandleMany`:, it returns the `[]*router.Routes`.
// Usage:
// func (*Controller) BeforeActivation(b mvc.BeforeActivation) {
// 	b.HandleMany("GET", "/path /path1" /path2", "HandlePath")
// }
func (c *ControllerActivator) HandleMany(method, path, funcName string, middleware ...context.Handler) []*router.Route {
	return c.handleMany(method, path, funcName, true, middleware...)
}

func (c *ControllerActivator) handleMany(method, path, funcName string, override bool, middleware ...context.Handler) []*router.Route {
	if method == "" || path == "" || funcName == "" ||
		c.isReservedMethod(funcName) {
		// isReservedMethod -> if it's already registered
		// by a previous Handle or analyze methods internally.
		return nil
	}

	// get the method from the controller type.
	m, ok := c.Type.MethodByName(funcName)
	if !ok {
		c.addErr(fmt.Errorf("MVC: function '%s' doesn't exist inside the '%s' controller",
			funcName, c.fullName))
		return nil
	}

	// parse a route template which contains the parameters organised.
	tmpl, err := macro.Parse(path, c.macros)
	if err != nil {
		c.addErr(fmt.Errorf("MVC: fail to parse the path for '%s.%s': %v", c.fullName, funcName, err))
		return nil
	}

	// get the function's input.
	funcIn := getInputArgsFromFunc(m.Type)
	// get the path parameters bindings from the template,
	// use the function's input except the receiver which is the
	// end-dev's controller pointer.
	pathParams := getPathParamsForInput(c.tmplParamStartIndex, tmpl.Params, funcIn[1:]...)
	// get the function's input arguments' bindings.
	funcDependencies := c.dependencies.Clone()
	funcDependencies.AddValues(pathParams...)

	handler := c.handlerOf(m, funcDependencies)

	// register the handler now.
	routes := c.router.HandleMany(method, path, append(middleware, handler)...)
	if routes == nil {
		c.addErr(fmt.Errorf("MVC: unable to register a route for the path for '%s.%s'", c.fullName, funcName))
		return nil
	}

	for _, r := range routes {
		// change the main handler's name in order to respect the controller's and give
		// a proper debug message.
		r.MainHandlerName = fmt.Sprintf("%s.%s", c.fullName, funcName)
	}

	// add this as a reserved method name in order to
	// be sure that the same route
	// (method is allowed to be registered more than one on different routes - v11.2).

	existingRoutes, exist := c.routes[funcName]
	if override || !exist {
		c.routes[funcName] = routes
	} else {
		c.routes[funcName] = append(existingRoutes, routes...)
	}

	return routes
}

var emptyIn = []reflect.Value{}

func (c *ControllerActivator) attachInjector() {
	if c.injector == nil {
		c.injector = di.MakeStructInjector(
			di.ValueOf(c.Value),
			c.sorter,
			di.Values(c.dependencies).CloneWithFieldsOf(c.Value)...,
		)
		// c.injector = di.Struct(c.Value, c.dependencies...)
		if !c.servesWebsocket {
			golog.Debugf("MVC Controller [%s] [Scope=%s]", c.fullName, c.injector.Scope)
		} else {
			golog.Debugf("MVC Websocket Controller [%s]", c.fullName)
		}

		if c.injector.Has {
			golog.Debugf("Dependencies:\n%s", c.injector.String())
		}
	}
}

func (c *ControllerActivator) handlerOf(m reflect.Method, funcDependencies []reflect.Value) context.Handler {
	// Remember:
	// The `Handle->handlerOf` can be called from `BeforeActivation` event
	// then, the c.injector is nil because
	// we may not have the dependencies bind-ed yet.
	// To solve this we're doing a check on the FIRST `Handle`,
	// if c.injector is nil, then set it with the current bindings,
	// these bindings can change after, so first add dependencies and after register routes.
	c.attachInjector()

	// fmt.Printf("for %s | values: %s\n", funcName, funcDependencies)
	funcInjector := di.Func(m.Func, funcDependencies...)
	funcInjector.ErrorHandler = c.errorHandler

	// fmt.Printf("actual injector's inputs length: %d\n", funcInjector.Length)
	if funcInjector.Has {
		golog.Debugf("MVC dependencies of method '%s.%s':\n%s", c.fullName, m.Name, funcInjector.String())
	}

	var (
		implementsBase         = isBaseController(c.Type)
		implementsErrorHandler = isErrorHandler(c.Type)
		hasBindableFields      = c.injector.CanInject
		hasBindableFuncInputs  = funcInjector.Has
		funcHasErrorOut        = hasErrorOutArgs(m)

		call = m.Func.Call
	)

	if !implementsBase && !hasBindableFields && !hasBindableFuncInputs && !implementsErrorHandler {
		return func(ctx context.Context) {
			hero.DispatchFuncResult(ctx, c.errorHandler, call(c.injector.AcquireSlice()))
		}
	}

	n := m.Type.NumIn()
	return func(ctx context.Context) {
		var (
			ctrl         = c.injector.Acquire()
			errorHandler = c.errorHandler
		)

		// inject struct fields first before the BeginRequest and EndRequest, if any,
		// in order to be able to have access there.
		if hasBindableFields {
			c.injector.InjectElem(ctx, ctrl.Elem())
		}

		// check if has BeginRequest & EndRequest, before try to bind the method's inputs.
		if implementsBase {
			// the Interface(). is faster than MethodByName or pre-selected methods.
			b := ctrl.Interface().(BaseController)
			// init the request.
			b.BeginRequest(ctx)

			// if begin request stopped the execution.
			if ctx.IsStopped() {
				return
			}

			defer b.EndRequest(ctx)
		}

		if funcHasErrorOut && implementsErrorHandler {
			errorHandler = ctrl.Interface().(di.ErrorHandler)
		}

		if hasBindableFuncInputs {
			// means that ctxValue is not initialized before by the controller's struct injector.

			in := make([]reflect.Value, n)
			in[0] = ctrl
			funcInjector.Inject(ctx, &in)

			if ctx.IsStopped() {
				return // stop as soon as possible, although it would stop later on if `ctx.StopExecution` called.
			}

			hero.DispatchFuncResult(ctx, errorHandler, call(in))
			return
		}

		hero.DispatchFuncResult(ctx, errorHandler, ctrl.Method(m.Index).Call(emptyIn))
	}
}
