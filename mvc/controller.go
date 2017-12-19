package mvc

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
	"github.com/kataras/iris/core/router/macro"
	"github.com/kataras/iris/mvc/di"

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
	Handle(method, path, funcName string, middleware ...context.Handler) *router.Route
}

type BeforeActivation interface {
	shared
	Dependencies() *di.Values
}

type AfterActivation interface {
	shared
	DependenciesReadOnly() di.ValuesReadOnly
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

	// the methods names that is already binded to a handler,
	// the BeginRequest, EndRequest and BeforeActivation are reserved by the internal implementation.
	reservedMethods []string

	// the bindings that comes from the Engine and the controller's filled fields if any.
	// Can be binded to the the new controller's fields and method that is fired
	// on incoming requests.
	dependencies di.Values

	// on activate.
	injector *di.StructInjector
}

func getNameOf(typ reflect.Type) string {
	elemTyp := di.IndirectType(typ)

	typName := elemTyp.Name()
	pkgPath := elemTyp.PkgPath()
	fullname := pkgPath[strings.LastIndexByte(pkgPath, '/')+1:] + "." + typName

	return fullname
}

func newControllerActivator(router router.Party, controller interface{}, dependencies di.Values) *ControllerActivator {
	var (
		val = reflect.ValueOf(controller)
		typ = val.Type()

		// the full name of the controller: its type including the package path.
		fullName = getNameOf(typ)
	)

	c := &ControllerActivator{
		// give access to the Router to the end-devs if they need it for some reason,
		// i.e register done handlers.
		router:   router,
		Value:    val,
		Type:     typ,
		fullName: fullName,
		// set some methods that end-dev cann't use accidentally
		// to register a route via the `Handle`,
		// all available exported and compatible methods
		// are being appended to the slice at the `parseMethods`,
		// if a new method is registered via `Handle` its function name
		// is also appended to that slice.
		//
		// TODO: now that BaseController is totally optionally
		// we have to check if BeginRequest and EndRequest should be here.
		reservedMethods: whatReservedMethods(typ),
		dependencies:    dependencies,
	}

	return c
}

func whatReservedMethods(typ reflect.Type) []string {
	methods := []string{"BeforeActivation", "AfterActivation"}
	if isBaseController(typ) {
		methods = append(methods, "BeginRequest", "EndRequest")
	}

	return methods
}

func (c *ControllerActivator) Dependencies() *di.Values {
	return &c.dependencies
}

func (c *ControllerActivator) DependenciesReadOnly() di.ValuesReadOnly {
	return c.dependencies
}

func (c *ControllerActivator) Name() string {
	return c.fullName
}

func (c *ControllerActivator) Router() router.Party {
	return c.router
}

// Singleton returns new if all incoming clients' requests
// have the same controller instance.
// This is done automatically by iris to reduce the creation
// of a new controller on each request, if the controller doesn't contain
// any unexported fields and all fields are services-like, static.
func (c *ControllerActivator) Singleton() bool {
	if c.injector == nil {
		panic("MVC: IsRequestScoped used on an invalid state the API gives access to it only `AfterActivation`, report this as bug")
	}
	return c.injector.State == di.Singleton
}

// checks if a method is already registered.
func (c *ControllerActivator) isReservedMethod(name string) bool {
	for _, s := range c.reservedMethods {
		if s == name {
			return true
		}
	}

	return false
}

func (c *ControllerActivator) parseMethod(m reflect.Method) {
	httpMethod, httpPath, err := parseMethod(m, c.isReservedMethod)
	if err != nil {
		if err != errSkip {
			err = fmt.Errorf("MVC: fail to parse the route path and HTTP method for '%s.%s': %v", c.fullName, m.Name, err)
			c.router.GetReporter().AddErr(err)

		}
		return
	}

	c.Handle(httpMethod, httpPath, m.Name)
}

// register all available, exported methods to handlers if possible.
func (c *ControllerActivator) parseMethods() {
	n := c.Type.NumMethod()
	for i := 0; i < n; i++ {
		m := c.Type.Method(i)
		c.parseMethod(m)
	}
}

func (c *ControllerActivator) activate() {
	c.parseMethods()
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

	// Remember:
	// we cannot simply do that and expect to work:
	// hasStructInjector = c.injector != nil && c.injector.Valid
	// hasFuncInjector   = funcInjector != nil && funcInjector.Valid
	// because
	// the `Handle` can be called from `BeforeActivation` callbacks
	// and before activation, the c.injector is nil because
	// we may not have the dependencies binded yet. But if `c.injector.Valid`
	// inside the Handelr works because it's set on the `activate()` method.
	// To solve this we can make check on the FIRST `Handle`,
	// if c.injector is nil, then set it with the current bindings,
	// so the user should bind the dependencies needed before the `Handle`
	// this is a logical flow, so we will choose that one ->
	if c.injector == nil {
		// first, set these bindings to the passed controller, they will be useless
		// if the struct contains any dynamic value because this controller will
		// be never fired as it's but we make that in order to get the length of the static
		// matched dependencies of the struct.
		c.injector = di.MakeStructInjector(c.Value, hijacker, typeChecker, c.dependencies...)
		if c.injector.HasFields {
			golog.Debugf("MVC dependencies of '%s':\n%s", c.fullName, c.injector.String())
		}
	}

	// get the method from the controller type.
	m, ok := c.Type.MethodByName(funcName)
	if !ok {
		err := fmt.Errorf("MVC: function '%s' doesn't exist inside the '%s' controller",
			funcName, c.fullName)
		c.router.GetReporter().AddErr(err)
		return nil
	}

	// parse a route template which contains the parameters organised.
	tmpl, err := macro.Parse(path, c.router.Macros())
	if err != nil {
		err = fmt.Errorf("MVC: fail to parse the path for '%s.%s': %v", c.fullName, funcName, err)
		c.router.GetReporter().AddErr(err)
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
	if route != nil {
		// change the main handler's name in order to respect the controller's and give
		// a proper debug message.
		route.MainHandlerName = fmt.Sprintf("%s.%s", c.fullName, funcName)

		// add this as a reserved method name in order to
		// be sure that the same func will not be registered again,
		// even if a custom .Handle later on.
		c.reservedMethods = append(c.reservedMethods, funcName)
	}

	return route
}

var emptyIn = []reflect.Value{}

func (c *ControllerActivator) handlerOf(m reflect.Method, funcDependencies []reflect.Value) context.Handler {
	// fmt.Printf("for %s | values: %s\n", funcName, funcDependencies)
	funcInjector := di.MakeFuncInjector(m.Func, hijacker, typeChecker, funcDependencies...)
	// fmt.Printf("actual injector's inputs length: %d\n", funcInjector.Length)
	if funcInjector.Valid {
		golog.Debugf("MVC dependencies of method '%s.%s':\n%s", c.fullName, m.Name, funcInjector.String())
	}

	var (
		implementsBase        = isBaseController(c.Type)
		hasBindableFields     = c.injector.CanInject
		hasBindableFuncInputs = funcInjector.Valid

		call = m.Func.Call
	)

	if !implementsBase && !hasBindableFields && !hasBindableFuncInputs {
		return func(ctx context.Context) {
			DispatchFuncResult(ctx, call(c.injector.NewAsSlice()))
		}
	}

	n := m.Type.NumIn()
	return func(ctx context.Context) {
		var (
			ctrl     = c.injector.New()
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
			DispatchFuncResult(ctx, call(in))
			return
		}

		DispatchFuncResult(ctx, ctrl.Method(m.Index).Call(emptyIn))
	}

}
