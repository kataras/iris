package persistence

import (
	"reflect"

	"github.com/kataras/iris/mvc/activator/field"
)

// Controller is responsible to load from the original
// end-developer's main controller's value
// and re-store the persistence data by scanning the original.
// It stores and sets to each new controller
// the optional data  that should be shared among all requests.
type Controller struct {
	fields []field.Field
}

// Load scans and load for persistence data based on the `iris:"persistence"` tag.
//
// The type is the controller's Type.
// the "val" is the original end-developer's controller's Value.
// Returns nil if no persistence data to store found.
func Load(typ reflect.Type, val reflect.Value) *Controller {
	matcher := func(elemField reflect.StructField) bool {
		if tag, ok := elemField.Tag.Lookup("iris"); ok {
			if tag == "persistence" {
				return true
			}
		}
		return false
	}

	handler := func(f *field.Field) {
		valF := val.Field(f.Index)
		if valF.IsValid() || (valF.Kind() == reflect.Ptr && !valF.IsNil()) {
			val := reflect.ValueOf(valF.Interface())
			if val.IsValid() || (val.Kind() == reflect.Ptr && !val.IsNil()) {
				f.Value = val
			}
		}
	}

	fields := field.LookupFields(typ.Elem(), matcher, handler)

	if len(fields) == 0 {
		return nil
	}

	return &Controller{
		fields: fields,
	}
}

// Handle re-stores the persistence data at the current controller.
func (pc *Controller) Handle(c reflect.Value) {
	elem := c.Elem() // controller should always be a pointer at this state
	for _, f := range pc.fields {
		f.SendTo(elem)
	}
}
