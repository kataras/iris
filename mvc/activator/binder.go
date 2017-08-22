package activator

import (
	"reflect"

	"github.com/kataras/iris/context"
)

type binder struct {
	values []interface{}
	fields []field

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

func (b *binder) lookup(elem reflect.Type) (fields []field) {
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

		for i, n := 0, elem.NumField(); i < n; i++ {
			elemField := elem.Field(i)
			if elemField.Type == value.Type() {
				// we area inside the correct type
				// println("[0] prepare bind filed for " + elemField.Name)
				fields = append(fields, field{
					Index: i,
					Name:  elemField.Name,
					Type:  elemField.Type,
					Value: value,
				})
				continue
			}

			f := lookupStruct(elemField.Type, value)
			if f != nil {
				fields = append(fields, field{
					Index:    i,
					Name:     elemField.Name,
					Type:     elemField.Type,
					embedded: f,
				})
			}

		}
	}
	return
}

func lookupStruct(elem reflect.Type, value reflect.Value) *field {
	// ignore if that field is not a struct
	if elem.Kind() != reflect.Struct {
		// and it's not a controller because we don't want to accidentally
		// set fields to other user fields. Or no?
		//  ||
		// 	(elem.Name() != "" && !strings.HasSuffix(elem.Name(), "Controller")) {
		return nil
	}

	// search by fields.
	for i, n := 0, elem.NumField(); i < n; i++ {
		elemField := elem.Field(i)
		if elemField.Type == value.Type() {
			// println("Types are equal of: " + elemField.Type.Name() + " " + elemField.Name + " and " + value.Type().Name())
			// we area inside the correct type.
			return &field{
				Index: i,
				Name:  elemField.Name,
				Type:  elemField.Type,
				Value: value,
			}
		}

		// if field is struct and the value is struct
		// then try inside its fields for a compatible
		// field type.
		if elemField.Type.Kind() == reflect.Struct && value.Type().Kind() == reflect.Struct {
			elemFieldEmb := elem.Field(i)
			f := lookupStruct(elemFieldEmb.Type, value)
			if f != nil {
				fp := &field{
					Index:    i,
					Name:     elemFieldEmb.Name,
					Type:     elemFieldEmb.Type,
					embedded: f,
				}
				return fp
			}
		}
	}
	return nil
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
		f.sendTo(elem)
	}
}
