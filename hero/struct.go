package hero

import (
	"fmt"
	"reflect"

	"github.com/kataras/iris/v12/context"
)

// Sorter is the type for sort customization of a struct's fields
// and its available bindable values.
//
// Sorting applies only when a field can accept more than one registered value.
type Sorter func(t1 reflect.Type, t2 reflect.Type) bool

// sortByNumMethods is a builtin sorter to sort fields and values
// based on their type and its number of methods, highest number of methods goes first.
//
// It is the default sorter on struct injector of `hero.Struct` method.
var sortByNumMethods Sorter = func(t1 reflect.Type, t2 reflect.Type) bool {
	if t1.Kind() != t2.Kind() {
		return true
	}

	if k := t1.Kind(); k == reflect.Interface || k == reflect.Struct {
		return t1.NumMethod() > t2.NumMethod()
	} else if k != reflect.Struct {
		return false // non-structs goes last.
	}

	return true
}

// Struct keeps a record of a particular struct value injection.
// See `Container.Struct` and `mvc#Application.Handle` methods.
type Struct struct {
	ptrType     reflect.Type
	ptrValue    reflect.Value // the original ptr struct value.
	elementType reflect.Type  // the original struct type.
	bindings    []*binding    // struct field bindings.

	Container *Container
	Singleton bool
}

type singletonStruct interface {
	Singleton() bool
}

func isMarkedAsSingleton(structPtr any) bool {
	if sing, ok := structPtr.(singletonStruct); ok && sing.Singleton() {
		return true
	}

	return false
}

func makeStruct(structPtr interface{}, c *Container, partyParamsCount int) *Struct {
	v := valueOf(structPtr)
	typ := v.Type()
	if typ.Kind() != reflect.Ptr || indirectType(typ).Kind() != reflect.Struct {
		panic("binder: struct: should be a pointer to a struct value")
	}

	isSingleton := isMarkedAsSingleton(structPtr)

	disablePayloadAutoBinding := c.DisablePayloadAutoBinding
	enableStructDependents := c.EnableStructDependents
	disableStructDynamicBindings := c.DisableStructDynamicBindings
	if isSingleton {
		disablePayloadAutoBinding = true
		enableStructDependents = false
		disableStructDynamicBindings = true
	}

	// get struct's fields bindings.
	bindings := getBindingsForStruct(v, c.Dependencies, c.MarkExportedFieldsAsRequired, disablePayloadAutoBinding, enableStructDependents, c.DependencyMatcher, partyParamsCount, c.Sorter)

	// length bindings of 0, means that it has no fields or all mapped deps are static.
	// If static then Struct.Acquire will return the same "value" instance, otherwise it will create a new one.
	singleton := true
	elem := v.Elem()

	// fmt.Printf("Service: %s, Bindings(%d):\n", typ, len(bindings))
	for _, b := range bindings {
		// fmt.Printf("* " + b.String() + "\n")
		if b.Dependency.Static {
			// Fill now.
			input, err := b.Dependency.Handle(nil, b.Input)
			if err != nil {
				if err == ErrSeeOther {
					continue
				}

				panic(err)
			}

			elem.FieldByIndex(b.Input.StructFieldIndex).Set(input)
		} else if !b.Dependency.Static {
			if disableStructDynamicBindings {
				panic(fmt.Sprintf("binder: DisableStructDynamicBindings setting is set to true: dynamic binding found: %s", b.String()))
			}

			singleton = false
		}
	}

	if isSingleton && !singleton {
		panic(fmt.Sprintf("binder: Singleton setting is set to true but struct has dynamic bindings: %s", typ))
	}

	s := &Struct{
		ptrValue:    v,
		ptrType:     typ,
		elementType: elem.Type(),
		bindings:    bindings,
		Singleton:   singleton,
	}

	isErrHandler := isErrorHandler(typ)
	newContainer := c.Clone()
	newContainer.fillReport(typ.String(), bindings)
	// Add the controller dependency itself as func dependency but with a known type which should be explicit binding
	// in order to keep its maximum priority.
	newContainer.Register(s.Acquire).Explicitly().DestType = typ

	newContainer.GetErrorHandler = func(ctx *context.Context) ErrorHandler {
		if isErrHandler {
			return ctx.Controller().Interface().(ErrorHandler)
		}

		return c.GetErrorHandler(ctx)
	}

	s.Container = newContainer
	return s
}

// Acquire returns a struct value based on the request.
// If the dependencies are all static then these are already set-ed at the initialization of this Struct
// and the same struct value instance will be returned, ignoring the Context. Otherwise
// a new struct value with filled fields by its pre-calculated bindings will be returned instead.
func (s *Struct) Acquire(ctx *context.Context) (reflect.Value, error) {
	if s.Singleton {
		ctx.Values().Set(context.ControllerContextKey, s.ptrValue)
		return s.ptrValue, nil
	}

	ctrl := ctx.Controller()
	if ctrl.Kind() == reflect.Invalid ||
		ctrl.Type() != s.ptrType /* in case of changing controller in the same request (see RouteOverlap feature) */ {
		ctrl = reflect.New(s.elementType)
		ctx.Values().Set(context.ControllerContextKey, ctrl)
		elem := ctrl.Elem()
		for _, b := range s.bindings {
			input, err := b.Dependency.Handle(ctx, b.Input)
			if err != nil {
				if err == ErrSeeOther {
					continue
				}

				s.Container.GetErrorHandler(ctx).HandleError(ctx, err)

				if ctx.IsStopped() {
					// return emptyValue, err
					return ctrl, err
				} // #1629
			}

			elem.FieldByIndex(b.Input.StructFieldIndex).Set(input)
		}
	}

	return ctrl, nil
}

// MethodHandler accepts a "methodName" that should be a valid an exported
// method of the struct and returns its converted Handler.
//
// Second input is optional,
// even zero is a valid value and can resolve path parameters correctly if from root party.
func (s *Struct) MethodHandler(methodName string, paramsCount int) context.Handler {
	m, ok := s.ptrValue.Type().MethodByName(methodName)
	if !ok {
		panic(fmt.Sprintf("struct: method: %s does not exist", methodName))
	}

	return makeHandler(m.Func, s.Container, paramsCount)
}
