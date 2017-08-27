package model

import (
	"reflect"

	"github.com/kataras/iris/mvc/activator/field"

	"github.com/kataras/iris/context"
)

// Controller is responsible
// to load and handle the `Model(s)` inside a controller struct
// via the  `iris:"model"` tag field.
// It stores the optional models from
// the struct's fields values that
// are being setted by the method function
// and set them as ViewData.
type Controller struct {
	fields []field.Field
}

// Load tries to lookup and set for any valid model field.
// Returns nil if no models are being used.
func Load(typ reflect.Type) *Controller {
	matcher := func(f reflect.StructField) bool {
		if tag, ok := f.Tag.Lookup("iris"); ok {
			if tag == "model" {
				return true
			}
		}
		return false
	}

	fields := field.LookupFields(typ.Elem(), matcher, nil)

	if len(fields) == 0 {
		return nil
	}

	mc := &Controller{
		fields: fields,
	}
	return mc
}

// Handle transfer the models to the view.
func (mc *Controller) Handle(ctx context.Context, c reflect.Value) {
	elem := c.Elem() // controller should always be a pointer at this state

	for _, f := range mc.fields {

		index := f.GetIndex()
		typ := f.GetType()
		name := f.GetTagName()

		elemField := elem.FieldByIndex(index)
		// check if current controller's element field
		// is valid, is not nil and it's type is the same (should be but make that check to be sure).
		if !elemField.IsValid() ||
			(elemField.Kind() == reflect.Ptr && elemField.IsNil()) ||
			elemField.Type() != typ {
			continue
		}

		fieldValue := elemField.Interface()
		ctx.ViewData(name, fieldValue)
		// /*maybe some time in the future*/ if resetable {
		// 	// clean up
		// 	elemField.Set(reflect.Zero(typ))
		// }

	}
}
