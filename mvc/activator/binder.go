package activator

import (
	"reflect"

	"github.com/kataras/iris/mvc/activator/field"

	"github.com/kataras/iris/context"
)

// binder accepts a value of something
// and tries to find its equalivent type
// inside the controller and sets that to it,
// after that each new instance of the controller will have
// this value on the specific field, like persistence data control does.

type binder struct {
	elemType reflect.Type
	// values and fields are matched on the `match`.
	values []interface{}
	fields []field.Field

	// saves any middleware that may need to be passed to the router,
	// statically, to gain performance.
	middleware context.Handlers
}

func (b *binder) bind(value interface{}) {
	if value == nil {
		return
	}

	b.values = append(b.values, value) // keep values.

	b.match(value)
}

func (b *binder) isEmpty() bool {
	// if nothing valid found return nil, so the caller
	// can omit the binder.
	if len(b.fields) == 0 && len(b.middleware) == 0 {
		return true
	}

	return false
}

func (b *binder) storeValueIfMiddleware(value reflect.Value) bool {
	if value.CanInterface() {
		if m, ok := value.Interface().(context.Handler); ok {
			b.middleware = append(b.middleware, m)
			return true
		}
		if m, ok := value.Interface().(func(context.Context)); ok {
			b.middleware = append(b.middleware, m)
			return true
		}
	}
	return false
}

func (b *binder) match(v interface{}) {
	value := reflect.ValueOf(v)
	// handlers will be recognised as middleware, not struct fields.
	// End-Developer has the option to call any handler inside
	// the controller's `BeginRequest` and `EndRequest`, the
	// state is respected from the method handler already.
	if b.storeValueIfMiddleware(value) {
		// stored as middleware, continue to the next field, we don't have
		// to bind anything here.
		return
	}

	matcher := func(elemField reflect.StructField) bool {
		// If the controller's field is interface then check
		// if the given binded value implements that interface.
		// i.e MovieController { Service services.MovieService /* interface */ }
		// app.Controller("/", new(MovieController),
		// 	services.NewMovieMemoryService(...))
		//
		// `services.NewMovieMemoryService` returns a `*MovieMemoryService`
		// that implements the `MovieService` interface.
		if elemField.Type.Kind() == reflect.Interface {
			return value.Type().Implements(elemField.Type)
		}
		return elemField.Type == value.Type()
	}

	handler := func(f *field.Field) {
		f.Value = value
	}

	b.fields = append(b.fields, field.LookupFields(b.elemType, matcher, handler)...)
}

func (b *binder) handle(c reflect.Value) {
	// we could make check for middlewares here but
	// these could easly be used outside of the controller
	// so we don't have to initialize a controller to call them
	// so they don't belong actually here, we will register them to the
	// router itself, before the controller's handler to gain performance,
	// look `activator.go#RegisterMethodHandlers` for more.

	elem := c.Elem() // controller should always be a pointer at this state
	for _, f := range b.fields {
		f.SendTo(elem)
	}
}
