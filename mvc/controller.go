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
	IsRequestScoped() bool
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

		// the full name of the controller, it's its type including the package path.
		fullName = getNameOf(typ)
	)

	// add the manual filled fields to the dependencies.
	filledFieldValues := di.LookupNonZeroFieldsValues(val)
	dependencies.AddValue(filledFieldValues...)

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

// IsRequestScoped returns new if each request has its own instance
// of the controller and it contains dependencies that are not manually
// filled by the struct initialization from the caller.
func (c *ControllerActivator) IsRequestScoped() bool {
	// if the c.injector == nil means that is not seted to invalidate state,
	// so it contains more fields that are filled by the end-dev.
	// This "strange" check happens because the `IsRequestScoped` may
	// called before the controller activation complete its task (see Handle: if c.injector == nil).
	if c.injector == nil { // is nil so it contains more fields, maybe request-scoped or dependencies.
		return true
	}
	if c.injector.Valid {
		// if injector is not nil:
		// if it is !Valid then all fields are manually filled by the end-dev (see `newControllerActivator`).
		// if it is Valid then it's filled on the first `Handle`
		// and it has more than one valid dependency from the registered values.
		return true
	}
	// it's not nil and it's !Valid.
	return false
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
	//	wg := &sync.WaitGroup{}
	// wg.Add(n)
	for i := 0; i < n; i++ {
		m := c.Type.Method(i)
		c.parseMethod(m)
	}
	// wg.Wait()
}

func (c *ControllerActivator) activate() {
	c.parseMethods()
}

var emptyIn = []reflect.Value{}

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

	// add this as a reserved method name in order to
	// be sure that the same func will not be registered again, even if a custom .Handle later on.
	c.reservedMethods = append(c.reservedMethods, funcName)

	// get the function's input.
	funcIn := getInputArgsFromFunc(m.Type)

	// get the path parameters bindings from the template,
	// use the function's input except the receiver which is the
	// end-dev's controller pointer.
	pathParams := getPathParamsForInput(tmpl.Params, funcIn[1:]...)
	// get the function's input arguments' bindings.
	funcDependencies := c.dependencies.Clone()
	funcDependencies.AddValue(pathParams...)

	// fmt.Printf("for %s | values: %s\n", funcName, funcDependencies)
	funcInjector := di.MakeFuncInjector(m.Func, hijacker, typeChecker, funcDependencies...)
	// fmt.Printf("actual injector's inputs length: %d\n", funcInjector.Length)

	// the element value, not the pointer, wil lbe used to create a
	// new controller on each incoming request.

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
		// check if manually filled + any dependencies are only static, if so
		// and the total struct's fields are equal these static dependencies length
		// then we don't need to create a new struct on each request.
		//
		// We use our custom NumFields here because the std "reflect" package
		// checks only for the current struct and not for embedded's exported fields.
		totalFieldsLength := di.NumFields(di.IndirectType(c.Type))

		// first, set these bindings to the passed controller, they will be useless
		// if the struct contains any dynamic value because this controller will
		// be never fired as it's but we make that in order to get the length of the static
		// matched dependencies of the struct.
		c.injector = di.MakeStructInjector(c.Value, hijacker, typeChecker, c.dependencies...)
		matchedStaticDependenciesLength := c.injector.InjectElemStaticOnly(di.IndirectValue(c.Value))

		if c.injector.Valid {
			golog.Debugf("MVC dependencies of '%s':\n%s", c.fullName, c.injector.String())
		}

		if matchedStaticDependenciesLength == totalFieldsLength {
			// all fields are filled by the end-developer or via static dependencies (if context is there then it will be filled by the MakeStructInjector so we don't worry about it),
			// the controller doesn't contain any other field neither any dynamic binding as well.
			// Therefore we don't need to create a new controller each time.
			// Set the c.injector now instead on the first `Handle` and set it to invalid state
			// in order to `buildControllerHandler` ignore the
			// creation of a new controller value on each incoming request.
			c.injector = &di.StructInjector{Valid: false}
		}
	}

	if funcInjector.Valid {
		golog.Debugf("MVC dependencies of method '%s.%s':\n%s", c.fullName, funcName, funcInjector.String())
	}

	handler := buildControllerHandler(m, c.Type, c.Value, c.injector, funcInjector, funcIn)

	// register the handler now.
	route := c.router.Handle(method, path, append(middleware, handler)...)
	if route != nil {
		// change the main handler's name in order to respect the controller's and give
		// a proper debug message.
		route.MainHandlerName = fmt.Sprintf("%s.%s", c.fullName, funcName)
	}

	return route
}

// buildControllerHandler has many many dublications but we do that to achieve the best
// performance possible, to use the information we know
// and calculate what is needed and what not in serve-time.
func buildControllerHandler(m reflect.Method, typ reflect.Type, initRef reflect.Value, structInjector *di.StructInjector, funcInjector *di.FuncInjector, funcIn []reflect.Type) context.Handler {
	var (
		hasStructInjector = structInjector != nil && structInjector.Valid
		hasFuncInjector   = funcInjector != nil && funcInjector.Valid

		implementsBase = isBaseController(typ)
		// we will make use of 'n' to make a slice of reflect.Value
		// to pass into if the function has input arguments that
		// are will being filled by the funcDependencies.
		n = len(funcIn)

		elemTyp = di.IndirectType(typ)
	)

	// if it doesn't implement the base controller,
	// it may have struct injector and/or func injector.
	if !implementsBase {

		if !hasStructInjector {
			// if the controller doesn't have a struct injector
			// and the controller's fields are empty or all set-ed by the end-dev
			// then we don't need a new controller instance, we use the passed controller instance.

			if !hasFuncInjector {
				return func(ctx context.Context) {
					DispatchFuncResult(ctx, initRef.Method(m.Index).Call(emptyIn))
				}
			}

			return func(ctx context.Context) {
				in := make([]reflect.Value, n, n)
				in[0] = initRef
				funcInjector.Inject(&in, reflect.ValueOf(ctx))
				if ctx.IsStopped() {
					return
				}

				DispatchFuncResult(ctx, m.Func.Call(in))
			}
		}

		// it has struct injector for sure and maybe a func injector.
		if !hasFuncInjector {
			return func(ctx context.Context) {
				ctrl := reflect.New(elemTyp)
				ctxValue := reflect.ValueOf(ctx)
				elem := ctrl.Elem()
				structInjector.InjectElem(elem, ctxValue)
				if ctx.IsStopped() {
					return
				}

				DispatchFuncResult(ctx, ctrl.Method(m.Index).Call(emptyIn))
			}
		}

		// has struct injector and func injector.
		return func(ctx context.Context) {
			ctrl := reflect.New(elemTyp)
			ctxValue := reflect.ValueOf(ctx)

			elem := ctrl.Elem()
			structInjector.InjectElem(elem, ctxValue)
			if ctx.IsStopped() {
				return
			}

			in := make([]reflect.Value, n, n)
			in[0] = ctrl
			funcInjector.Inject(&in, ctxValue)
			if ctx.IsStopped() {
				return
			}

			DispatchFuncResult(ctx, m.Func.Call(in))
		}

	}

	// if implements the base controller,
	// it may have struct injector and func injector as well.
	return func(ctx context.Context) {
		ctrl := reflect.New(elemTyp)

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

		if !hasStructInjector && !hasFuncInjector {
			DispatchFuncResult(ctx, ctrl.Method(m.Index).Call(emptyIn))
		} else {
			ctxValue := reflect.ValueOf(ctx)
			if hasStructInjector {
				elem := ctrl.Elem()
				structInjector.InjectElem(elem, ctxValue)
				if ctx.IsStopped() {
					return
				}

				// we do this in order to reduce in := make...
				// if not func input binders, we execute the handler with empty input args.
				if !hasFuncInjector {
					DispatchFuncResult(ctx, ctrl.Method(m.Index).Call(emptyIn))
				}
			}
			// otherwise, it has one or more valid input binders,
			// make the input and call the func using those.
			if hasFuncInjector {
				in := make([]reflect.Value, n, n)
				in[0] = ctrl
				funcInjector.Inject(&in, ctxValue)
				if ctx.IsStopped() {
					return
				}

				DispatchFuncResult(ctx, m.Func.Call(in))
			}

		}
	}
}
