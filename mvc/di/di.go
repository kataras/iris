package di

import "reflect"

type (
	// Hijacker is a type which is used to catch fields or function's input argument
	// to bind a custom object based on their type.
	Hijacker func(reflect.Type) (*BindObject, bool)
	// TypeChecker checks if a specific field's or function input argument's
	// is valid to be binded.
	TypeChecker func(reflect.Type) bool
)

// D is the Dependency Injection container,
// it contains the Values that can be changed before the injectors.
// `Struct` and the `Func` methods returns an injector for specific
// struct instance-value or function.
type D struct {
	Values

	hijacker Hijacker
	goodFunc TypeChecker
}

// New creates and returns a new Dependency Injection container.
// See `Values` field and `Func` and `Struct` methods for more.
func New() *D {
	return &D{}
}

// Hijack sets a hijacker function, read the `Hijacker` type for more explaination.
func (d *D) Hijack(fn Hijacker) *D {
	d.hijacker = fn
	return d
}

// GoodFunc sets a type checker for a valid function that can be binded,
// read the `TypeChecker` type for more explaination.
func (d *D) GoodFunc(fn TypeChecker) *D {
	d.goodFunc = fn
	return d
}

// Clone returns a new Dependency Injection container, it adopts the
// parent's (current "D") hijacker, good func type checker and all dependencies values.
func (d *D) Clone() *D {
	clone := New()
	clone.hijacker = d.hijacker
	clone.goodFunc = d.goodFunc

	// copy the current dynamic bindings (func binders)
	// and static struct bindings (services) to this new child.
	if n := len(d.Values); n > 0 {
		values := make(Values, n, n)
		copy(values, d.Values)
		clone.Values = values
	}

	return clone
}

// Struct is being used to return a new injector based on
// a struct value instance, if it contains fields that the types of those
// are matching with one or more of the `Values` then they are binded
// with the injector's `Inject` and `InjectElem` methods.
func (d *D) Struct(s interface{}) *StructInjector {
	if s == nil {
		return nil
	}
	v := ValueOf(s)

	return MakeStructInjector(
		v,
		d.hijacker,
		d.goodFunc,
		d.Values...,
	)
}

// Func is being used to return a new injector based on
// a function, if it contains input arguments that the types of those
// are matching with one or more of the `Values` then they are binded
// to the function's input argument when called
// with the injector's `Fill` method.
func (d *D) Func(fn interface{}) *FuncInjector {
	return MakeFuncInjector(
		ValueOf(fn),
		d.hijacker,
		d.goodFunc,
		d.Values...,
	)
}
