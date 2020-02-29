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

type Struct struct {
	ptrType     reflect.Type
	ptrValue    reflect.Value // the original ptr struct value.
	elementType reflect.Type  // the original struct type.
	bindings    []*Binding    // struct field bindings.

	Container *Container
	Singleton bool
}

func makeStruct(structPtr interface{}, c *Container) *Struct {
	v := valueOf(structPtr)
	typ := v.Type()
	if typ.Kind() != reflect.Ptr || indirectType(typ).Kind() != reflect.Struct {
		panic("binder: struct: should be a pointer to a struct value")
	}

	// get struct's fields bindings.
	bindings := getBindingsForStruct(v, c.Dependencies, c.ParamStartIndex, c.Sorter)

	// length bindings of 0, means that it has no fields or all mapped deps are static.
	// If static then Struct.Acquire will return the same "value" instance, otherwise it will create a new one.
	singleton := true
	elem := v.Elem()
	for _, b := range bindings {
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
			singleton = false
		}
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
	// Add the controller dependency itself as func dependency but with a known type which should be explicit binding
	// in order to keep its maximum priority.
	newContainer.Register(s.Acquire).
		Explicitly().
		DestType = typ

	newContainer.GetErrorHandler = func(ctx context.Context) ErrorHandler {
		if isErrHandler {
			return ctx.Controller().Interface().(ErrorHandler)
		}

		return c.GetErrorHandler(ctx)
	}

	s.Container = newContainer
	return s
}

func (s *Struct) Acquire(ctx context.Context) (reflect.Value, error) {
	if s.Singleton {
		ctx.Values().Set(context.ControllerContextKey, s.ptrValue)
		return s.ptrValue, nil
	}

	ctrl := ctx.Controller()
	if ctrl.Kind() == reflect.Invalid {
		ctrl = reflect.New(s.elementType)
		ctx.Values().Set(context.ControllerContextKey, ctrl)
		elem := ctrl.Elem()
		for _, b := range s.bindings {
			input, err := b.Dependency.Handle(ctx, b.Input)
			if err != nil {
				if err == ErrSeeOther {
					continue
				}

				// return emptyValue, err
				return ctrl, err
			}
			elem.FieldByIndex(b.Input.StructFieldIndex).Set(input)
		}
	}

	return ctrl, nil
}

func (s *Struct) MethodHandler(methodName string) context.Handler {
	m, ok := s.ptrValue.Type().MethodByName(methodName)
	if !ok {
		panic(fmt.Sprintf("struct: method: %s does not exist", methodName))
	}

	return makeHandler(m.Func, s.Container)
}
