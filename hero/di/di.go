// Package di provides dependency injection for the Iris Hero and Iris MVC new features.
// It's used internally by "hero" and "mvc" packages.
package di

import (
	"reflect"

	"github.com/kataras/iris/v12/context"
)

type (
	// ErrorHandler is the optional interface to handle errors per hero func,
	// see `mvc/Application#HandleError` for MVC application-level error handler registration too.
	//
	// Handles non-nil errors return from a hero handler or a controller's method (see `DispatchFuncResult`)
	// and (from v12.1.8) the error may return from a request-scoped dynamic dependency (see `MakeReturnValue`).
	ErrorHandler interface {
		HandleError(ctx context.Context, err error)
	}

	// ErrorHandlerFunc implements the `ErrorHandler`.
	// It describes the type defnition for an error handler.
	ErrorHandlerFunc func(ctx context.Context, err error)
)

// HandleError fires when the `DispatchFuncResult` or `MakereturnValue` return a non-nil error.
func (fn ErrorHandlerFunc) HandleError(ctx context.Context, err error) {
	fn(ctx, err)
}

// DefaultErrorHandler is the default error handler will be fired on
// any error from registering a request-scoped dynamic dependency and on a controller's method failure.
var DefaultErrorHandler ErrorHandler = ErrorHandlerFunc(func(ctx context.Context, err error) {
	if err == nil {
		return
	}

	ctx.StatusCode(400)
	ctx.WriteString(err.Error())
	ctx.StopExecution()
})

var emptyValue reflect.Value

// DefaultFallbackBinder used to bind any oprhan inputs. Its error is handled by the `ErrorHandler`.
var DefaultFallbackBinder FallbackBinder = func(ctx context.Context, input OrphanInput) (newValue reflect.Value, err error) {
	wasPtr := input.Type.Kind() == reflect.Ptr

	newValue = reflect.New(IndirectType(input.Type))
	ptr := newValue.Interface()

	switch ctx.GetContentTypeRequested() {
	case context.ContentXMLHeaderValue:
		err = ctx.ReadXML(ptr)
	case context.ContentYAMLHeaderValue:
		err = ctx.ReadYAML(ptr)
	case context.ContentFormHeaderValue:
		err = ctx.ReadQuery(ptr)
	case context.ContentFormMultipartHeaderValue:
		err = ctx.ReadForm(ptr)
	default:
		err = ctx.ReadJSON(ptr)
		// json
	}

	// if err != nil {
	// 	return emptyValue, err
	// }

	if !wasPtr {
		newValue = newValue.Elem()
	}

	return newValue, err
}

// Struct is being used to return a new injector based on
// a struct value instance, if it contains fields that the types of those
// are matching with one or more of the `Values` then they are binded
// with the injector's `Inject` and `InjectElem` methods.
func Struct(s interface{}, values ...reflect.Value) *StructInjector {
	if s == nil {
		return &StructInjector{}
	}

	return MakeStructInjector(
		ValueOf(s),
		SortByNumMethods,
		Values(values).CloneWithFieldsOf(s)...,
	)
}

// Func is being used to return a new injector based on
// a function, if it contains input arguments that the types of those
// are matching with one or more of the `Values` then they are binded
// to the function's input argument when called
// with the injector's `Inject` method.
func Func(fn interface{}, values ...reflect.Value) *FuncInjector {
	if fn == nil {
		return &FuncInjector{}
	}

	return MakeFuncInjector(
		ValueOf(fn),
		values...,
	)
}

// D is the Dependency Injection container,
// it contains the Values that can be changed before the injectors.
// `Struct` and the `Func` methods returns an injector for specific
// struct instance-value or function.
type D struct {
	Values

	fallbackBinder FallbackBinder
	errorHandler   ErrorHandler
	sorter         Sorter
}

// OrphanInput represents an input without registered dependency.
// Used to help the framework (or the caller) auto-resolve it by the request.
type OrphanInput struct {
	// Index int // function or struct field index.
	Type reflect.Type
}

// FallbackBinder represents a handler of oprhan input values, handler's input arguments or controller's fields.
type FallbackBinder func(ctx context.Context, input OrphanInput) (reflect.Value, error)

// New creates and returns a new Dependency Injection container.
// See `Values` field and `Func` and `Struct` methods for more.
func New() *D {
	return &D{
		errorHandler:   DefaultErrorHandler,
		fallbackBinder: DefaultFallbackBinder,
	}
}

// FallbackBinder adds a binder which will handle any oprhan input values.
// See `FallbackBinder` type.
func (d *D) FallbackBinder(fallbackBinder FallbackBinder) *D {
	d.fallbackBinder = fallbackBinder
	return d
}

// ErrorHandler adds a handler which will be fired when a handler's second output argument is error and it's not nil
// or when a request-scoped dynamic function dependency's second output argument is error and it's not nil.
func (d *D) ErrorHandler(errorHandler ErrorHandler) *D {
	d.errorHandler = errorHandler
	return d
}

// Sort sets the fields and valid bindable values sorter for struct injection.
func (d *D) Sort(with Sorter) *D {
	d.sorter = with
	return d
}

// Clone returns a new Dependency Injection container, it adopts the
// parent's (current "D") hijacker, good func type checker, sorter and all dependencies values.
func (d *D) Clone() *D {
	return &D{
		Values:         d.Values.Clone(),
		fallbackBinder: d.fallbackBinder,
		errorHandler:   d.errorHandler,
		sorter:         d.sorter,
	}
}

// Struct is being used to return a new injector based on
// a struct value instance, if it contains fields that the types of those
// are matching with one or more of the `Values` then they are binded
// with the injector's `Inject` and `InjectElem` methods.
func (d *D) Struct(s interface{}) *StructInjector {
	if s == nil {
		return &StructInjector{}
	}

	injector := MakeStructInjector(
		ValueOf(s),
		d.sorter,
		d.Values.CloneWithFieldsOf(s)...,
	)

	injector.ErrorHandler = d.errorHandler
	injector.FallbackBinder = d.fallbackBinder

	return injector
}

// Func is being used to return a new injector based on
// a function, if it contains input arguments that the types of those
// are matching with one or more of the `Values` then they are binded
// to the function's input argument when called
// with the injector's `Inject` method.
func (d *D) Func(fn interface{}) *FuncInjector {
	if fn == nil {
		return &FuncInjector{}
	}

	injector := MakeFuncInjector(
		ValueOf(fn),
		d.Values...,
	)

	injector.ErrorHandler = d.errorHandler
	injector.FallbackBinder = d.fallbackBinder

	return injector
}
