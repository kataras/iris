package activator

import (
	"reflect"

	"github.com/kataras/iris/mvc/activator/field"

	"github.com/kataras/iris/context"
)

type binder struct {
	values []interface{}
	fields []field.Field

	// saves any middleware that may need to be passed to the router,
	// statically, to gain performance.
	middleware context.Handlers
}

// binder accepts a value of something
// and tries to find its equalivent type
// inside the controller and sets that to it,
// after that each new instance of the controller will have
// this value on the specific field, like persistence data control does.
//
// returns a nil binder if values are not valid bindable data to the controller type.
func newBinder(elemType reflect.Type, values []interface{}) *binder {
	if len(values) == 0 {
		return nil
	}

	b := &binder{values: values}
	b.fields = b.lookup(elemType)

	// if nothing valid found return nil, so the caller
	// can omit the binder.
	if len(b.fields) == 0 && len(b.middleware) == 0 {
		return nil
	}

	return b
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

func (b *binder) lookup(elem reflect.Type) (fields []field.Field) {
	for _, v := range b.values {
		value := reflect.ValueOf(v)
		// handlers will be recognised as middleware, not struct fields.
		// End-Developer has the option to call any handler inside
		// the controller's `BeginRequest` and `EndRequest`, the
		// state is respected from the method handler already.
		if b.storeValueIfMiddleware(value) {
			// stored as middleware, continue to the next field, we don't have
			// to bind anything here.
			continue
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

		fields = append(fields, field.LookupFields(elem, matcher, handler)...)
	}
	return
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
