package activator

import (
	"reflect"
	"strings"

	"github.com/kataras/iris/mvc/activator/methodfunc"
	"github.com/kataras/iris/mvc/activator/model"
	"github.com/kataras/iris/mvc/activator/persistence"

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
		// The name of the front controller struct.
		Name string
		// FullName it's the last package path segment + "." + the Name.
		// i.e: if login-example/user/controller.go, the FullName is "user.Controller".
		FullName string
		// the type of the user/dev's "c" controller (interface{}).
		Type reflect.Type
		// it's the first passed value of the controller instance,
		// we need this to collect and save the persistence fields' values.
		Value reflect.Value

		binder                *binder // executed even before the BeginRequest if not nil.
		modelController       *model.Controller
		persistenceController *persistence.Controller
	}
)

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
func ActivateController(base BaseController, bindValues []interface{}) (TController, error) {

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
	pkgPath := val.Type().PkgPath()
	fullName := pkgPath[strings.LastIndexByte(pkgPath, '/')+1:] + "." + ctrlName

	// set the binder, can be nil this check at made at runtime.
	binder := newBinder(typ.Elem(), bindValues)
	if binder != nil {
		for _, bf := range binder.fields {
			golog.Debugf("MVC %s: binder loaded for '%s' with value:\n%#v",
				fullName, bf.GetFullName(), bf.GetValue())
		}
	}

	t := TController{
		Name:                  ctrlName,
		FullName:              fullName,
		Type:                  typ,
		Value:                 val,
		binder:                binder,
		modelController:       model.Load(typ),
		persistenceController: persistence.Load(typ, val),
	}

	return t, nil
}

// HandlerOf builds the handler for a type based on the specific method func.
func (t TController) HandlerOf(methodFunc methodfunc.MethodFunc) context.Handler {
	var (
		// shared, per-controller
		elem     = t.Type.Elem()
		ctrlName = t.Name

		hasPersistenceData = t.persistenceController != nil
		hasModels          = t.modelController != nil
		// per-handler
		handleRequest = methodFunc.MethodCall
	)

	return func(ctx context.Context) {
		// create a new controller instance of that type(>ptr).
		c := reflect.New(elem)
		if t.binder != nil {
			t.binder.handle(c)
		}

		b := c.Interface().(BaseController)
		b.SetName(ctrlName)

		// if has persistence data then set them
		// before the end-developer's handler in order to be available there.
		if hasPersistenceData {
			t.persistenceController.Handle(c)
		}

		// init the request.
		b.BeginRequest(ctx)
		if ctx.IsStopped() {
			return
		}

		// the most important, execute the specific function
		// from the controller that is responsible to handle
		// this request, by method and path.
		handleRequest(ctx, c.Method(methodFunc.Index))
		// if had models, set them after the end-developer's handler.
		if hasModels {
			t.modelController.Handle(ctx, c)
		}

		// finally, execute the controller, don't check for IsStopped.
		b.EndRequest(ctx)
	}
}

// RegisterFunc used by the caller to register the result routes.
type RegisterFunc func(relPath string, httpMethod string, handler ...context.Handler)

// RegisterMethodHandlers receives a `TController`, description of the
// user's controller, and calls the "registerFunc" for each of its
// method handlers.
//
// Not useful for the end-developer, but may needed for debugging
// at the future.
func RegisterMethodHandlers(t TController, registerFunc RegisterFunc) {
	var middleware context.Handlers

	if t.binder != nil {
		if m := t.binder.middleware; len(m) > 0 {
			middleware = m
		}
	}
	// the actual method functions
	// i.e for "GET" it's the `Get()`.
	methods := methodfunc.Resolve(t.Type)

	// range over the type info's method funcs,
	// build a new handler for each of these
	// methods and register them to their
	// http methods using the registerFunc, which is
	// responsible to convert these into routes
	// and add them to router via the APIBuilder.
	for _, m := range methods {
		h := t.HandlerOf(m)
		if h == nil {
			golog.Debugf("MVC %s: nil method handler found for %s", t.FullName, m.Name)
			continue
		}
		registeredHandlers := append(middleware, h)
		registerFunc(m.RelPath, m.HTTPMethod, registeredHandlers...)

		golog.Debugf("MVC %s: %s %s maps to function[%d] '%s'", t.FullName,
			m.HTTPMethod,
			m.RelPath,
			m.Index,
			m.Name)
	}
}

// Register receives a "controller",
// a pointer of an instance which embeds the `Controller`,
// the value of "baseControllerFieldName" should be `Controller`.
func Register(controller BaseController, bindValues []interface{},
	registerFunc RegisterFunc) error {

	t, err := ActivateController(controller, bindValues)
	if err != nil {
		return err
	}

	RegisterMethodHandlers(t, registerFunc)
	return nil
}
