package mvc2

import (
	"fmt"
	"reflect"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
	"github.com/kataras/iris/core/router/macro"
)

type BaseController interface {
	BeginRequest(context.Context)
	EndRequest(context.Context)
}

// C is the basic BaseController type that can be used as an embedded anonymous field
// to custom end-dev controllers.
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
// Look `core/router#APIBuilder#Controller` method too.
//
// It completes the `activator.BaseController` interface.
//
// Example at: https://github.com/kataras/iris/tree/master/_examples/mvc/overview/web/controllers.
// Example usage at: https://github.com/kataras/iris/blob/master/mvc/method_result_test.go#L17.
type C struct {
	// The current context.Context.
	//
	// we have to name it for two reasons:
	// 1: can't ignore these via reflection, it doesn't give an option to
	// see if the functions is derived from another type.
	// 2: end-developer may want to use some method functions
	// or any fields that could be conflict with the context's.
	Ctx context.Context
}

var _ BaseController = &C{}

// BeginRequest starts the request by initializing the `Context` field.
func (c *C) BeginRequest(ctx context.Context) { c.Ctx = ctx }

// EndRequest does nothing, is here to complete the `BaseController` interface.
func (c *C) EndRequest(ctx context.Context) {}

type ControllerActivator struct {
	Engine *Engine
	// the router is used on the `Activate` and can be used by end-dev on the `OnActivate`
	// to register any custom controller's functions as handlers but we will need it here
	// in order to not create a new type like `ActivationPayload` for the `OnActivate`.
	Router router.Party

	// initRef BaseController // the BaseController as it's passed from the end-dev.
	Value reflect.Value // the BaseController's Value.
	Type  reflect.Type  // raw type of the BaseController (initRef).
	// FullName it's the last package path segment + "." + the Name.
	// i.e: if login-example/user/controller.go, the FullName is "user.Controller".
	FullName string

	// the methods names that is already binded to a handler,
	// the BeginRequest, EndRequest and OnActivate are reserved by the internal implementation.
	reservedMethods []string

	// input are always empty after the `activate`
	// are used to build the bindings, and we need this field
	// because we have 3 states (Engine.Input, OnActivate, Bind)
	// that we can add or override binding values.
	input []reflect.Value

	// the bindings that comes from input (and Engine) and can be binded to the controller's(initRef) fields.
	bindings *targetStruct
}

func newControllerActivator(router router.Party, controller BaseController, bindValues ...reflect.Value) *ControllerActivator {
	// the following will make sure that if
	// the controller's has set-ed pointer struct fields by the end-dev
	// we will include them to the bindings.
	// set bindings to the non-zero pointer fields' values that may be set-ed by
	// the end-developer when declaring the controller,
	// activate listeners needs them in order to know if something set-ed already or not,
	// look `BindTypeExists`.

	var (
		val = reflect.ValueOf(controller)
		typ = val.Type()

		fullName = getNameOf(typ)
	)

	c := &ControllerActivator{
		Router:   router,
		Value:    val,
		Type:     typ,
		FullName: fullName,
		reservedMethods: []string{
			"BeginRequest",
			"EndRequest",
			"OnActivate",
		},
		input: append(lookupNonZeroFieldsValues(val), bindValues...),
	}

	c.parseMethods()
	return c
}

func (c *ControllerActivator) isReservedMethod(name string) bool {
	for _, s := range c.reservedMethods {
		if s == name {
			return true
		}
	}

	return false
}

func (c *ControllerActivator) parseMethods() {
	// register all available, exported methods to handlers if possible.
	n := c.Type.NumMethod()
	for i := 0; i < n; i++ {
		m := c.Type.Method(i)

		httpMethod, httpPath, err := parseMethod(m, c.isReservedMethod)
		if err != nil {
			if err != errSkip {
				err = fmt.Errorf("MVC: fail to parse the route path and HTTP method for '%s.%s': %v", c.FullName, m.Name, err)
				c.Router.GetReporter().AddErr(err)

			}
			continue
		}

		c.Handle(httpMethod, httpPath, m.Name)
	}
}

// SetBindings will override any bindings with the new "values".
func (c *ControllerActivator) SetBindings(values ...reflect.Value) {
	// set field index with matching binders, if any.
	c.input = values
	c.bindings = newTargetStruct(c.Value, values...)
}

// Bind binds values to this controller, if you want to share
// binding values between controllers use the Engine's `Bind` function instead.
func (c *ControllerActivator) Bind(values ...interface{}) {
	for _, val := range values {
		if v := reflect.ValueOf(val); goodVal(v) {
			c.input = append(c.input, v)
		}
	}
}

// BindTypeExists returns true if a binder responsible to
// bind and return a type of "typ" is already registered to this controller.
func (c *ControllerActivator) BindTypeExists(typ reflect.Type) bool {
	for _, in := range c.input {
		if equalTypes(in.Type(), typ) {
			return true
		}
	}
	return false
}

func (c *ControllerActivator) activate() {
	c.SetBindings(c.input...)
}

var emptyIn = []reflect.Value{}

func (c *ControllerActivator) Handle(method, path, funcName string, middleware ...context.Handler) error {
	if method == "" || path == "" || funcName == "" ||
		c.isReservedMethod(funcName) {
		// isReservedMethod -> if it's already registered
		// by a previous Handle or analyze methods internally.
		return errSkip
	}

	m, ok := c.Type.MethodByName(funcName)
	if !ok {
		err := fmt.Errorf("MVC: function '%s' doesn't exist inside the '%s' controller",
			funcName, c.FullName)
		c.Router.GetReporter().AddErr(err)
		return err
	}

	tmpl, err := macro.Parse(path, c.Router.Macros())
	if err != nil {
		err = fmt.Errorf("MVC: fail to parse the path for '%s.%s': %v", c.FullName, funcName, err)
		c.Router.GetReporter().AddErr(err)
		return err
	}

	// add this as a reserved method name in order to
	// be sure that the same func will not be registered again, even if a custom .Handle later on.
	c.reservedMethods = append(c.reservedMethods, funcName)

	// fmt.Printf("===============%s.%s==============\n", c.FullName, funcName)

	funcIn := getInputArgsFromFunc(m.Type) // except the receiver, which is the controller pointer itself.

	pathParams := getPathParamsForInput(tmpl.Params, funcIn[1:]...)
	funcBindings := newTargetFunc(m.Func, pathParams...)

	elemTyp := indirectTyp(c.Type) // the element value, not the pointer.

	n := len(funcIn)

	handler := func(ctx context.Context) {

		// create a new controller instance of that type(>ptr).
		ctrl := reflect.New(elemTyp)
		b := ctrl.Interface().(BaseController) // the Interface(). is faster than MethodByName or pre-selected methods.
		// init the request.
		b.BeginRequest(ctx)

		// if begin request stopped the execution.
		if ctx.IsStopped() {
			return
		}

		if !c.bindings.Valid && !funcBindings.Valid {
			DispatchFuncResult(ctx, ctrl.Method(m.Index).Call(emptyIn))
		} else {
			ctxValue := reflect.ValueOf(ctx)

			if c.bindings.Valid {
				elem := ctrl.Elem()
				c.bindings.Fill(elem, ctxValue)
				if ctx.IsStopped() {
					return
				}

				// we do this in order to reduce in := make...
				// if not func input binders, we execute the handler with empty input args.
				if !funcBindings.Valid {
					DispatchFuncResult(ctx, ctrl.Method(m.Index).Call(emptyIn))
				}
			}
			// otherwise, it has one or more valid input binders,
			// make the input and call the func using those.
			if funcBindings.Valid {
				in := make([]reflect.Value, n, n)
				in[0] = ctrl
				funcBindings.Fill(&in, ctxValue)
				if ctx.IsStopped() {
					return
				}

				DispatchFuncResult(ctx, m.Func.Call(in))
			}

		}

		// end the request, don't check for stopped because this does the actual writing
		// if no response written already.
		b.EndRequest(ctx)
	}

	// register the handler now.
	c.Router.Handle(method, path, append(middleware, handler)...).
		// change the main handler's name in order to respect the controller's and give
		// a proper debug message.
		MainHandlerName = fmt.Sprintf("%s.%s", c.FullName, funcName)

	return nil
}
