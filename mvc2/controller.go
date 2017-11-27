package mvc2

import (
// "reflect"

// "github.com/kataras/golog"
// "github.com/kataras/iris/context"
// // "github.com/kataras/iris/core/router"
// "github.com/kataras/iris/mvc/activator"
// "github.com/kataras/iris/mvc/activator/methodfunc"
)

// no, we will not make any changes to the controller's implementation
// let's no re-write the godlike code I wrote two months ago
// , just improve it by implementing the only one missing feature:
// bind/map/handle custom controller's functions to a custom router path
// like regexed.
//
// // BaseController is the interface that all controllers should implement.
// type BaseController interface {
// 	BeginRequest(ctx context.Context)
// 	EndRequest(ctx context.Context)
// }

// // type ControllerInitializer interface {
// // 	Init(r router.Party)
// // }

// // type activator struct {
// // 	Router router.Party
// // 	container *Mvc
// // }

// func registerController(m *Mvc, r router.Party, c BaseController) {

// }

// // ControllerHandler is responsible to dynamically bind a controller's functions
// // to the controller's http mechanism, can be used on the controller's `OnActivate` event.
// func ControllerHandler(controller activator.BaseController, funcName string) context.Handler {
// 	// we use funcName instead of an interface{} which can be safely binded with something like:
// 	// myController.HandleThis because we want to make sure that the end-developer
// 	// will make use a function of that controller that owns it because if not then
// 	// the BeginRequest and EndRequest will be called from other handler and also
// 	// the first input argument, which should be the controller itself may not be binded
// 	// to the current controller, all that are solved if end-dev knows what to do
// 	// but we can't bet on it.

// 	cVal := reflect.ValueOf(controller)
// 	elemTyp := reflect.TypeOf(controller) // with the pointer.
// 	m, exists := elemTyp.MethodByName(funcName)
// 	if !exists {
// 		golog.Errorf("mvc controller handler: function '%s' doesn't exist inside the '%s' controller",
// 			funcName, elemTyp.String())
// 		return nil
// 	}

// 	fn := cVal.MethodByName(funcName)
// 	if !fn.IsValid() {
// 		golog.Errorf("mvc controller handler: function '%s' inside the '%s' controller has not a valid value",
// 			funcName, elemTyp.String())
// 		return nil
// 	}

// 	info, ok := methodfunc.FetchFuncInfo(m)
// 	if !ok {
// 		golog.Errorf("mvc controller handler: could not resolve the func info from '%s'", funcName)
// 		return nil
// 	}

// 	methodFunc, err := methodfunc.ResolveMethodFunc(info)
// 	if err != nil {
// 		golog.Errorf("mvc controller handler: %v", err)
// 		return nil
// 	}

// 	m := New()
// 	m.In(controller) // bind the controller itself?
// 	/// TODO: first we must enable interface{} to be used as 'servetime input binder'
// 	// because it will try to match the type and add to its input if the
// 	// func input is that, and this binder will be available to every handler after that,
// 	// so it will be included to its 'in'.
// 	// MakeFuncInputBinder(func(ctx context.Context) interface{} {

// 	// 	// job here.

// 	// 	return controller
// 	// })

// 	h := m.Handler(fn.Interface())
// 	return func(ctx context.Context) {
// 		controller.BeginRequest(ctx)
// 		h(ctx)
// 		controller.EndRequest(ctx)
// 	}
// }
