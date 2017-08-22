package activator

import (
	"reflect"

	"github.com/kataras/iris/context"
)

type modelControl struct {
	fields []field
}

func (mc *modelControl) Load(t *TController) error {
	matcher := func(f reflect.StructField) bool {
		if tag, ok := f.Tag.Lookup("iris"); ok {
			if tag == "model" {
				return true
			}
		}
		return false
	}

	fields := lookupFields(t.Type.Elem(), matcher, nil)

	if len(fields) == 0 {
		// first is the `Controller` so we need to
		// check the second and after that.
		return ErrControlSkip
	}

	mc.fields = fields
	return nil
}

func (mc *modelControl) Handle(ctx context.Context, c reflect.Value, methodFunc func()) {
	elem := c.Elem() // controller should always be a pointer at this state

	for _, f := range mc.fields {

		index := f.getIndex()
		typ := f.getType()
		name := f.getTagName()

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
	}
}

// ModelControl returns a TControl which is responsible
// to load and handle the `Model(s)` inside a controller struct
// via the  `iris:"model"` tag field.
func ModelControl() TControl {
	return &modelControl{}
}
