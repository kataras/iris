package activator

import (
	"reflect"
	"strings"

	"github.com/kataras/iris/core/router/macro"
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

		valuePtr reflect.Value
		// // Methods and handlers, available after the Activate, can be seted `OnActivate` event as well.
		// Methods []methodfunc.MethodFunc

		Router RegisterFunc

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
	// the passed "c" instnace is not a valid type of `Controller` or `C`.
	ErrMissingControllerInstance = errors.New("controller should have a field of mvc.Controller or mvc.C type")
	// ErrInvalidControllerType fired when the "Controller" field is not
	// the correct type.
	ErrInvalidControllerType = errors.New("controller instance is not a valid implementation")
)

// BaseController is the controller interface,
// which the main request `Controller` will implement automatically.
// End-User doesn't need to have any knowledge of this if she/he doesn't want to implement
// a new Controller type.
// Controller looks the whole flow as one handler, so `ctx.Next`
// inside `BeginRequest` is not be respected.
// Alternative way to check if a middleware was procceed successfully
// and called its `ctx.Next` is the `ctx.Proceed(handler) bool`.
// You have to navigate to the `context/context#Proceed` function's documentation.
type BaseController interface {
	SetName(name string)
	BeginRequest(ctx context.Context)
	EndRequest(ctx context.Context)
}

// ActivateController returns a new controller type info description.
func newController(base BaseController, router RegisterFunc) (*TController, error) {
	// get and save the type.
	typ := reflect.TypeOf(base)
	if typ.Kind() != reflect.Ptr {
		typ = reflect.PtrTo(typ)
	}

	valPointer := reflect.ValueOf(base) // or value raw

	// first instance value, needed to validate
	// the actual type of the controller field
	// and to collect and save the instance's persistence fields'
	// values later on.
	val := reflect.Indirect(valPointer)

	ctrlName := val.Type().Name()
	pkgPath := val.Type().PkgPath()
	fullName := pkgPath[strings.LastIndexByte(pkgPath, '/')+1:] + "." + ctrlName

	t := &TController{
		Name:                  ctrlName,
		FullName:              fullName,
		Type:                  typ,
		Value:                 val,
		valuePtr:              valPointer,
		Router:                router,
		binder:                &binder{elemType: typ.Elem()},
		modelController:       model.Load(typ),
		persistenceController: persistence.Load(typ, val),
	}

	return t, nil
}

// BindValueTypeExists returns true if at least one type of "bindValue"
// is already binded to this `TController`.
func (t *TController) BindValueTypeExists(bindValue interface{}) bool {
	valueTyp := reflect.TypeOf(bindValue)
	for _, bindedValue := range t.binder.values {
		// type already exists, remember: binding here is per-type.
		if typ := reflect.TypeOf(bindedValue); typ == valueTyp ||
			(valueTyp.Kind() == reflect.Interface && typ.Implements(valueTyp)) {
			return true
		}
	}

	return false
}

// BindValue binds a value to a controller's field when request is served.
func (t *TController) BindValue(bindValues ...interface{}) {
	for _, bindValue := range bindValues {
		t.binder.bind(bindValue)
	}
}

// HandlerOf builds the handler for a type based on the specific method func.
func (t *TController) HandlerOf(methodFunc methodfunc.MethodFunc) context.Handler {
	var (
		// shared, per-controller
		elem      = t.Type.Elem()
		ctrlName  = t.Name
		hasBinder = !t.binder.isEmpty()

		hasPersistenceData = t.persistenceController != nil
		hasModels          = t.modelController != nil
		// per-handler
		handleRequest = methodFunc.MethodCall
	)

	return func(ctx context.Context) {
		// create a new controller instance of that type(>ptr).
		c := reflect.New(elem)
		if hasBinder {
			t.binder.handle(c)
		}

		b := c.Interface().(BaseController)
		b.SetName(ctrlName)

		// if has persistence data then set them
		// before the end-developer's handler in order to be available there.
		if hasPersistenceData {
			t.persistenceController.Handle(c)
		}

		// if previous (binded) handlers stopped the execution
		// we should know that.
		if ctx.IsStopped() {
			return
		}

		// init the request.
		b.BeginRequest(ctx)
		if ctx.IsStopped() { // if begin request stopped the execution
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

		// end the request, don't check for stopped because this does the actual writing
		// if no response written already.
		b.EndRequest(ctx)
	}
}

func (t *TController) registerMethodFunc(m methodfunc.MethodFunc) {
	var middleware context.Handlers

	if !t.binder.isEmpty() {
		if m := t.binder.middleware; len(m) > 0 {
			middleware = m
		}
	}

	h := t.HandlerOf(m)
	if h == nil {
		golog.Warnf("MVC %s: nil method handler found for %s", t.FullName, m.Name)
		return
	}

	registeredHandlers := append(middleware, h)
	t.Router(m.HTTPMethod, m.RelPath, registeredHandlers...)

	golog.Debugf("MVC %s: %s %s maps to function[%d] '%s'", t.FullName,
		m.HTTPMethod,
		m.RelPath,
		m.Index,
		m.Name)
}

func (t *TController) resolveAndRegisterMethods() {
	// the actual method functions
	// i.e for "GET" it's the `Get()`.
	methods, err := methodfunc.Resolve(t.Type)
	if err != nil {
		golog.Errorf("MVC %s: %s", t.FullName, err.Error())
		return
	}
	// range over the type info's method funcs,
	// build a new handler for each of these
	// methods and register them to their
	// http methods using the registerFunc, which is
	// responsible to convert these into routes
	// and add them to router via the APIBuilder.
	for _, m := range methods {
		t.registerMethodFunc(m)
	}
}

// Handle registers a method func but with a custom http method and relative route's path,
// it respects the rest of the controller's rules and guidelines.
func (t *TController) Handle(httpMethod, path, handlerFuncName string) bool {
	cTyp := t.Type // with the pointer.
	m, exists := cTyp.MethodByName(handlerFuncName)
	if !exists {
		golog.Errorf("MVC: function '%s' doesn't exist inside the '%s' controller",
			handlerFuncName, t.FullName)
		return false
	}

	info := methodfunc.FuncInfo{
		Name:       m.Name,
		Trailing:   m.Name,
		Type:       m.Type,
		Index:      m.Index,
		HTTPMethod: httpMethod,
	}

	tmpl, err := macro.Parse(path, macro.NewMap())
	if err != nil {
		golog.Errorf("MVC: fail to parse the path for '%s.%s': %v", t.FullName, handlerFuncName, err)
		return false
	}

	paramKeys := make([]string, len(tmpl.Params), len(tmpl.Params))
	for i, param := range tmpl.Params {
		paramKeys[i] = param.Name
	}

	methodFunc, err := methodfunc.ResolveMethodFunc(info, paramKeys...)
	if err != nil {
		golog.Errorf("MVC: function '%s' inside the '%s' controller: %v", handlerFuncName, t.FullName, err)
		return false
	}

	methodFunc.RelPath = path

	t.registerMethodFunc(methodFunc)
	return true
}

// func (t *TController) getMethodFuncByName(funcName string) (methodfunc.MethodFunc, bool) {
// 	cVal := t.Value
// 	cTyp := t.Type // with the pointer.
// 	m, exists := cTyp.MethodByName(funcName)
// 	if !exists {
// 		golog.Errorf("MVC: function '%s' doesn't exist inside the '%s' controller",
// 			funcName, cTyp.String())
// 		return methodfunc.MethodFunc{}, false
// 	}

// 	fn := cVal.MethodByName(funcName)
// 	if !fn.IsValid() {
// 		golog.Errorf("MVC: function '%s' inside the '%s' controller has not a valid value",
// 			funcName, cTyp.String())
// 		return methodfunc.MethodFunc{}, false
// 	}

// 	info, ok := methodfunc.FetchFuncInfo(m)
// 	if !ok {
// 		golog.Errorf("MVC: could not resolve the func info from '%s'", funcName)
// 		return methodfunc.MethodFunc{}, false
// 	}

// 	methodFunc, err := methodfunc.ResolveMethodFunc(info)
// 	if err != nil {
// 		golog.Errorf("MVC: %v", err)
// 		return methodfunc.MethodFunc{}, false
// 	}

// 	return methodFunc, true
// }

// // RegisterName registers a function by its name
// func (t *TController) RegisterName(funcName string) bool {
// 	methodFunc, ok := t.getMethodFuncByName(funcName)
// 	if !ok {
// 		return false
// 	}
// 	t.registerMethodFunc(methodFunc)
// 	return true
// }

// RegisterFunc used by the caller to register the result routes.
type RegisterFunc func(httpMethod string, relPath string, handler ...context.Handler)

// Register receives a "controller",
// a pointer of an instance which embeds the `Controller`,
// the value of "baseControllerFieldName" should be `Controller`.
func Register(controller BaseController, bindValues []interface{},
	registerFunc RegisterFunc) error {

	t, err := newController(controller, registerFunc)
	if err != nil {
		return err
	}

	t.BindValue(bindValues...)

	CallOnActivate(controller, t)

	for _, bf := range t.binder.fields {
		golog.Debugf("MVC %s: binder loaded for '%s' with value:\n%#v",
			t.FullName, bf.GetFullName(), bf.GetValue())
	}

	t.resolveAndRegisterMethods()

	return nil
}
