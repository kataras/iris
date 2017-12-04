package mvc2

import (
	"errors"
	"reflect"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
)

var (
	errNil           = errors.New("nil")
	errBad           = errors.New("bad")
	errAlreadyExists = errors.New("already exists")
)

type Engine struct {
	binders []*InputBinder

	Input []reflect.Value
}

func New() *Engine {
	return new(Engine)
}

func (e *Engine) Child() *Engine {
	child := New()

	// copy the current parent's ctx func binders and services to this new child.
	// if l := len(e.binders); l > 0 {
	// 	binders := make([]*InputBinder, l, l)
	// 	copy(binders, e.binders)
	// 	child.binders = binders
	// }
	if l := len(e.Input); l > 0 {
		input := make([]reflect.Value, l, l)
		copy(input, e.Input)
		child.Input = input
	}
	return child
}

func (e *Engine) Bind(binders ...interface{}) *Engine {
	for _, binder := range binders {
		// typ := resolveBinderType(binder)

		// var (
		// 	b   *InputBinder
		// 	err error
		// )

		// if typ == functionType {
		// 	b, err = MakeFuncInputBinder(binder)
		// } else if typ == serviceType {
		// 	b, err = MakeServiceInputBinder(binder)
		// } else {
		// 	err = errBad
		// }

		// if err != nil {
		// 	continue
		// }

		// e.binders = append(e.binders, b)

		e.Input = append(e.Input, reflect.ValueOf(binder))
	}

	return e
}

// BindTypeExists returns true if a binder responsible to
// bind and return a type of "typ" is already registered.
func (e *Engine) BindTypeExists(typ reflect.Type) bool {
	// for _, b := range e.binders {
	// 	if equalTypes(b.BindType, typ) {
	// 		return true
	// 	}
	// }
	for _, in := range e.Input {
		if equalTypes(in.Type(), typ) {
			return true
		}
	}
	return false
}

func (e *Engine) Handler(handler interface{}) context.Handler {
	h, _ := MakeHandler(handler, e.binders) // it logs errors already, so on any error the "h" will be nil.
	return h
}

type ActivateListener interface {
	OnActivate(*ControllerActivator)
}

func (e *Engine) Controller(router router.Party, controller BaseController) {
	ca := newControllerActivator(e, router, controller)
	if al, ok := controller.(ActivateListener); ok {
		al.OnActivate(ca)
	}
}
