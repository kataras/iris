package activator

import (
	"reflect"

	"github.com/kataras/golog"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/errors"
)

type (
	// TController is the type of the controller,
	// it contains all the necessary information to load
	// and serve the controller to the outside world,
	// think it as a "supervisor" of your Controller which
	// cares about you.
	TController struct {
		// the type of the user/dev's "c" controller (interface{}).
		Type reflect.Type
		// it's the first passed value of the controller instance,
		// we need this to collect and save the persistence fields' values.
		Value reflect.Value

		binder *binder // executed even before the BeginRequest if not nil.

		controls []TControl // executed on request, after the BeginRequest and before the EndRequest.

		// the actual method functions
		// i.e for "GET" it's the `Get()`
		//
		// Here we have a strange relation by-design.
		// It contains the methods
		// but we have different handlers
		// for each of these methods,
		// while in the same time all of these
		// are depend from this TypeInfo struct.
		// So we have TypeInfo -> Methods -> Each(TypeInfo, Method.Index)
		// -> Handler for X HTTPMethod, see `Register`.
		Methods []MethodFunc
	}
	// MethodFunc is part of the `TController`,
	// it contains the index for a specific http method,
	// taken from user's controller struct.
	MethodFunc struct {
		Index      int
		HTTPMethod string
	}
)

// ErrControlSkip never shows up, used to determinate
// if a control's Load return error is critical or not,
// `ErrControlSkip` means that activation can continue
// and skip this control.
var ErrControlSkip = errors.New("skip control")

// TControl is an optional feature that an app can benefit
// by using its own custom controls to control the flow
// inside a controller, they are being registered per controller.
//
// Naming:
// I could find better name such as 'Control',
// but I can imagine the user's confusion about `Controller`
// and `Control` types, they are different but they may
// use that as embedded, so it can not start with the world "C..".
// The best name that shows the relation between this
// and the controller type info struct(TController) is the "TControl",
// `TController` is prepended with "T" for the same reasons, it's different
// than `Controller`, the TController is the "description" of the user's
// `Controller` embedded field.
type TControl interface { // or CoreControl?
	// Load should returns nil  if its `Handle`
	// should be called on serve time.
	//
	// if error is filled then controller info
	// is not created and that error is returned to the
	// high-level caller, but the `ErrControlSkip` can be used
	// to skip the control without breaking the rest of the registration.
	Load(t *TController) error
	// Handle executes the control.
	// It accepts the context, the new controller instance
	// and the specific methodFunc based on the request.
	Handle(ctx context.Context, controller reflect.Value, methodFunc func())
}

func isControlErr(err error) bool {
	if err != nil {
		if isSkipper(err) {
			return false
		}
		return true
	}

	return false
}

func isSkipper(err error) bool {
	if err != nil {
		if err.Error() == ErrControlSkip.Error() {
			return true
		}
	}
	return false
}

// the parent package should complete this "interface"
// it's not exported, so their functions
// but reflect doesn't care about it, so we are ok
// to compare the type of the base controller field
// with this "ctrl", see `buildTypeInfo` and `buildMethodHandler`.

var (
	// ErrMissingControllerInstance is a static error which fired from `Controller` when
	// the passed "c" instnace is not a valid type of `Controller`.
	ErrMissingControllerInstance = errors.New("controller should have a field of Controller type")
	// ErrInvalidControllerType fired when the "Controller" field is not
	// the correct type.
	ErrInvalidControllerType = errors.New("controller instance is not a valid implementation")
)

// BaseController is the controller interface,
// which the main request `Controller` will implement automatically.
// End-User doesn't need to have any knowledge of this if she/he doesn't want to implement
// a new Controller type.
type BaseController interface {
	SetName(name string)
	BeginRequest(ctx context.Context)
	EndRequest(ctx context.Context)
}

// ActivateController returns a new controller type info description.
// A  TController is not useful for the end-developer
// but it can be used for debugging.
func ActivateController(base BaseController, bindValues []interface{},
	controls []TControl) (TController, error) {

	// get and save the type.
	typ := reflect.TypeOf(base)
	if typ.Kind() != reflect.Ptr {
		typ = reflect.PtrTo(typ)
	}

	// first instance value, needed to validate
	// the actual type of the controller field
	// and to collect and save the instance's persistence fields'
	// values later on.
	val := reflect.Indirect(reflect.ValueOf(base))
	ctrlName := val.Type().Name()

	// set the binder, can be nil this check at made at runtime.
	binder := newBinder(typ.Elem(), bindValues)
	if binder != nil {
		for _, bf := range binder.fields {
			golog.Debugf("MVC %s: binder loaded for '%s' with value:\n%#v",
				ctrlName, bf.getFullName(), bf.getValue())
		}
	}

	t := TController{
		Type:   typ,
		Value:  val,
		binder: binder,
	}

	// first the custom controls,
	// after these, the persistence,
	// the method control
	// which can set the model and
	// last the model control.
	controls = append(controls, []TControl{
		// PersistenceDataControl stores the optional data
		// that will be shared among all requests.
		PersistenceDataControl(),
		// MethodControl is the actual method function
		// i.e for "GET" it's the `Get()` that will be
		// fired.
		MethodControl(),
		// ModelControl stores the optional models from
		// the struct's fields values that
		// are being setted by the method function
		// and set them as ViewData.
		ModelControl()}...)

	for _, control := range controls {
		err := control.Load(&t)
		// fail on first control error if not ErrControlSkip.
		if isControlErr(err) {
			return t, err
		}

		if isSkipper(err) {
			continue
		}

		golog.Debugf("MVC %s: succeed load of the %#v", ctrlName, control)
		t.controls = append(t.controls, control)
	}

	return t, nil
}

// builds the handler for a type based on the method index (i.e Get() -> [0], Post() -> [1]).
func buildMethodHandler(t TController, methodFuncIndex int) context.Handler {
	elem := t.Type.Elem()
	ctrlName := t.Value.Type().Name()
	/*
		// good idea, it speeds up the whole thing by ~1MB per 20MB at my personal
		// laptop but this way the Model for example which is not a persistence
		// variable can stay for the next request
		// (if pointer receiver but if not then variables like `Tmpl` cannot stay)
		// and that will have unexpected results.
		// however we keep it here I want to see it every day in order to find a better way.

		type runtimeC struct {
			method func()
			c      reflect.Value
			elem   reflect.Value
			b      BaseController
		}

		pool := sync.Pool{
			New: func() interface{} {

				c := reflect.New(elem)
				methodFunc := c.Method(methodFuncIndex).Interface().(func())
				b, _ := c.Interface().(BaseController)

				elem := c.Elem()
				if t.binder != nil {
					t.binder.handle(elem)
				}

				rc := runtimeC{
					c:      c,
					elem:   elem,
					b:      b,
					method: methodFunc,
				}
				return rc
			},
		}
	*/

	return func(ctx context.Context) {
		// // create a new controller instance of that type(>ptr).
		c := reflect.New(elem)

		if t.binder != nil {
			t.binder.handle(c)
			if ctx.IsStopped() {
				return
			}
		}

		// get the Controller embedded field's addr.
		// it should never be invalid here because we made that checks on activation.
		// but if somone tries to "crack" that, then just stop the world in order to be notified,
		// we don't want to go away from that type of mistake.
		b := c.Interface().(BaseController)
		b.SetName(ctrlName)

		// init the request.
		b.BeginRequest(ctx)

		methodFunc := c.Method(methodFuncIndex).Interface().(func())
		// execute the controls by order, including the method control.
		for _, control := range t.controls {
			if ctx.IsStopped() {
				break
			}
			control.Handle(ctx, c, methodFunc)
		}

		// finally, execute the controller, don't check for IsStopped.
		b.EndRequest(ctx)
	}
}

// RegisterFunc used by the caller to register the result routes.
type RegisterFunc func(httpMethod string, handler ...context.Handler)

// RegisterMethodHandlers receives a `TController`, description of the
// user's controller, and calls the "registerFunc" for each of its
// method handlers.
//
// Not useful for the end-developer, but may needed for debugging
// at the future.
func RegisterMethodHandlers(t TController, registerFunc RegisterFunc) {
	// range over the type info's method funcs,
	// build a new handler for each of these
	// methods and register them to their
	// http methods using the registerFunc, which is
	// responsible to convert these into routes
	// and add them to router via the APIBuilder.

	var handlers context.Handlers

	if t.binder != nil {
		if m := t.binder.middleware; len(m) > 0 {
			handlers = append(handlers, t.binder.middleware...)
		}
	}

	for _, m := range t.Methods {
		methodHandler := buildMethodHandler(t, m.Index)
		registeredHandlers := append(handlers, methodHandler)
		registerFunc(m.HTTPMethod, registeredHandlers...)
	}
}

// Register receives a "controller",
// a pointer of an instance which embeds the `Controller`,
// the value of "baseControllerFieldName" should be `Controller`
// if embedded and "controls" that can intercept on controller
// activation and on the controller's handler, at serve-time.
func Register(controller BaseController, bindValues []interface{}, controls []TControl,
	registerFunc RegisterFunc) error {

	t, err := ActivateController(controller, bindValues, controls)
	if err != nil {
		return err
	}

	RegisterMethodHandlers(t, registerFunc)
	return nil
}
