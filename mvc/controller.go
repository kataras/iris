package mvc

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
	"github.com/kataras/iris/hero"
	"github.com/kataras/iris/hero/di"
	"github.com/kataras/iris/macro"

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
	Handle(httpMethod, path, funcName string, middleware ...context.Handler) *router.Route
}

// BeforeActivation is being used as the onle one input argument of a
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

// AfterActivation is being used as the onle one input argument of a
// `func(c *Controller) AfterActivation(a mvc.AfterActivation) {}`.
//
// It's being called after the `BeforeActivation`,
// and after controller's dependencies binded to the fields or the input arguments but before server ran.
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

	// initRef BaseController // the BaseController as it's passed from the end-dev.
	Value reflect.Value // the BaseController's Value.
	Type  reflect.Type  // raw type of the BaseController (initRef).
	// FullName it's the last package path segment + "." + the Name.
	// i.e: if login-example/user/controller.go, the FullName is "user.Controller".
	fullName string

	// the already-registered routes, key = the controller's function name.
	// End-devs can change some properties of the *Route on the `BeforeActivator` by using the
	// `GetRoute(functionName)`. It's a shield against duplications as well.
	routes map[string]*router.Route

	// the bindings that comes from the Engine and the controller's filled fields if any.
	// Can be binded to the the new controller's fields and method that is fired
	// on incoming requests.
	dependencies di.Values

	// initialized on the first `Handle`.
	injector *di.StructInjector
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

func newControllerActivator(router router.Party, controller interface{}, dependencies []reflect.Value) *ControllerActivator {
	typ := reflect.TypeOf(controller)

	c := &ControllerActivator{
		// give access to the Router to the end-devs if they need it for some reason,
		// i.e register done handlers.
		router: router,
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
	}

	return c
}

func whatReservedMethods(typ reflect.Type) map[string]*router.Route {
	methods := []string{"BeforeActivation", "AfterActivation"}
	//  BeforeActivatior/AfterActivation are not routes but they are
	// reserved names*
	if isBaseController(typ) {
		methods = append(methods, "BeginRequest", "EndRequest")
	}

	routes := make(map[string]*router.Route, len(methods))
	for _, m := range methods {
		routes[m] = &router.Route{}
	}

	return routes
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

// GetRoute returns a registered route based on the controller's method name.
// It can be used to change the route's name, which is useful for reverse routing
// inside views. Custom routes can be registered with `Handle`, which returns the *Route.
// This method exists mostly for the automatic method parsing based on the known patterns
// inside a controller.
//
// A check for `nil` is necessary for unregistered methods.
//
// See `Handle` too.
func (c *ControllerActivator) GetRoute(methodName string) *router.Route {
	for name, route := range c.routes {
		if name == methodName {
			return route
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
	return c.router.GetReporter().AddErr(err)
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
// Just like `APIBuilder`, it returns the `*router.Route`, if failed
// then it logs the errors and it returns nil, you can check the errors
// programmatically by the `APIBuilder#GetReporter`.
func (c *ControllerActivator) Handle(method, path, funcName string, middleware ...context.Handler) *router.Route {
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
	tmpl, err := macro.Parse(path, *c.router.Macros())
	if err != nil {
		c.addErr(fmt.Errorf("MVC: fail to parse the path for '%s.%s': %v", c.fullName, funcName, err))
		return nil
	}

	// get the function's input.
	funcIn := getInputArgsFromFunc(m.Type)
	// get the path parameters bindings from the template,
	// use the function's input except the receiver which is the
	// end-dev's controller pointer.
	pathParams := getPathParamsForInput(tmpl.Params, funcIn[1:]...)
	// get the function's input arguments' bindings.
	funcDependencies := c.dependencies.Clone()
	funcDependencies.AddValues(pathParams...)

	handler := c.handlerOf(m, funcDependencies)

	// register the handler now.
	route := c.router.Handle(method, path, append(middleware, handler)...)
	if route == nil {
		c.addErr(fmt.Errorf("MVC: unable to register a route for the path for '%s.%s'", c.fullName, funcName))
		return nil
	}

	// change the main handler's name in order to respect the controller's and give
	// a proper debug message.
	route.MainHandlerName = fmt.Sprintf("%s.%s", c.fullName, funcName)

	// add this as a reserved method name in order to
	// be sure that the same func will not be registered again,
	// even if a custom .Handle later on.
	c.routes[funcName] = route

	return route
}

var emptyIn = []reflect.Value{}

func (c *ControllerActivator) handlerOf(m reflect.Method, funcDependencies []reflect.Value) context.Handler {
	// Remember:
	// The `Handle->handlerOf` can be called from `BeforeActivation` event
	// then, the c.injector is nil because
	// we may not have the dependencies binded yet.
	// To solve this we're doing a check on the FIRST `Handle`,
	// if c.injector is nil, then set it with the current bindings,
	// these bindings can change after, so first add dependencies and after register routes.
	if c.injector == nil {
		c.injector = di.Struct(c.Value, c.dependencies...)
		if c.injector.Has {
			golog.Debugf("MVC dependencies of '%s':\n%s", c.fullName, c.injector.String())
		}
	}

	// fmt.Printf("for %s | values: %s\n", funcName, funcDependencies)

	funcInjector := di.Func(m.Func, funcDependencies...)
	// fmt.Printf("actual injector's inputs length: %d\n", funcInjector.Length)
	if funcInjector.Has {
		golog.Debugf("MVC dependencies of method '%s.%s':\n%s", c.fullName, m.Name, funcInjector.String())
	}

	var (
		implementsBase        = isBaseController(c.Type)
		hasBindableFields     = c.injector.CanInject
		hasBindableFuncInputs = funcInjector.Has

		call = m.Func.Call
	)

	if !implementsBase && !hasBindableFields && !hasBindableFuncInputs {
		return func(ctx context.Context) {
			hero.DispatchFuncResult(ctx, call(c.injector.AcquireSlice()))
		}
	}

	n := m.Type.NumIn()
	return func(ctx context.Context) {
		var (
			ctrl     = c.injector.Acquire()
			ctxValue reflect.Value
		)

		// inject struct fields first before the BeginRequest and EndRequest, if any,
		// in order to be able to have access there.
		if hasBindableFields {
			ctxValue = reflect.ValueOf(ctx)
			c.injector.InjectElem(ctrl.Elem(), ctxValue)
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

		if hasBindableFuncInputs {
			// means that ctxValue is not initialized before by the controller's struct injector.
			if !hasBindableFields {
				ctxValue = reflect.ValueOf(ctx)
			}

			in := make([]reflect.Value, n, n)
			in[0] = ctrl
			funcInjector.Inject(&in, ctxValue)

			// for idxx, inn := range in {
			// 	println("controller.go: execution: in.Value = "+inn.String()+" and in.Type = "+inn.Type().Kind().String()+" of index: ", idxx)
			// }

			hero.DispatchFuncResult(ctx, call(in))
			return
		}

		hero.DispatchFuncResult(ctx, ctrl.Method(m.Index).Call(emptyIn))
	}

}
