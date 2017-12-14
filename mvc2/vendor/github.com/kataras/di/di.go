package di

import "reflect"

type (
	Hijacker    func(reflect.Type) (*BindObject, bool)
	TypeChecker func(reflect.Type) bool
)

type D struct {
	Values

	hijacker Hijacker
	goodFunc TypeChecker
}

func New() *D {
	return &D{}
}

func (d *D) Hijack(fn Hijacker) *D {
	d.hijacker = fn
	return d
}

func (d *D) GoodFunc(fn TypeChecker) *D {
	d.goodFunc = fn
	return d
}

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

func (d *D) Struct(s interface{}) *StructInjector {
	if s == nil {
		return nil
	}
	v := valueOf(s)

	return MakeStructInjector(
		v,
		d.hijacker,
		d.goodFunc,
		d.Values...,
	)
}

func (d *D) Func(fn interface{}) *FuncInjector {
	return MakeFuncInjector(
		valueOf(fn),
		d.hijacker,
		d.goodFunc,
		d.Values...,
	)
}
