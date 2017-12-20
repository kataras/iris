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

// Hijack sets a hijacker function, read the `Hijacker` type for more explanation.
func (d *D) Hijack(fn Hijacker) *D {
	d.hijacker = fn
	return d
}

// GoodFunc sets a type checker for a valid function that can be binded,
// read the `TypeChecker` type for more explanation.
func (d *D) GoodFunc(fn TypeChecker) *D {
	d.goodFunc = fn
	return d
}

// Clone returns a new Dependency Injection container, it adopts the
// parent's (current "D") hijacker, good func type checker and all dependencies values.
func (d *D) Clone() *D {
	return &D{
		Values:   d.Values.Clone(),
		hijacker: d.hijacker,
		goodFunc: d.goodFunc,
	}
}

// Struct is being used to return a new injector based on
// a struct value instance, if it contains fields that the types of those
// are matching with one or more of the `Values` then they are binded
// with the injector's `Inject` and `InjectElem` methods.
func (d *D) Struct(s interface{}) *StructInjector {
	if s == nil {
		return &StructInjector{HasFields: false}
	}

	return MakeStructInjector(
		ValueOf(s),
		d.hijacker,
		d.goodFunc,
		d.Values.CloneWithFieldsOf(s)...,
	)
}

// Func is being used to return a new injector based on
// a function, if it contains input arguments that the types of those
// are matching with one or more of the `Values` then they are binded
// to the function's input argument when called
// with the injector's `Fill` method.
func (d *D) Func(fn interface{}) *FuncInjector {
	if fn == nil {
		return &FuncInjector{Valid: false}
	}

	return MakeFuncInjector(
		ValueOf(fn),
		d.hijacker,
		d.goodFunc,
		d.Values...,
	)
}
