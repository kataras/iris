package activator

import (
	"reflect"

	"github.com/kataras/iris/context"
)

type persistenceDataControl struct {
	fields []field
}

func (d *persistenceDataControl) Load(t *TController) error {
	matcher := func(elemField reflect.StructField) bool {
		if tag, ok := elemField.Tag.Lookup("iris"); ok {
			if tag == "persistence" {
				return true
			}
		}
		return false
	}

	handler := func(f *field) {
		valF := t.Value.Field(f.Index)
		if valF.IsValid() || (valF.Kind() == reflect.Ptr && !valF.IsNil()) {
			val := reflect.ValueOf(valF.Interface())
			if val.IsValid() || (val.Kind() == reflect.Ptr && !val.IsNil()) {
				f.Value = val
			}
		}
	}

	fields := lookupFields(t.Type.Elem(), matcher, handler)

	if len(fields) == 0 {
		// first is the `Controller` so we need to
		// check the second and after that.
		return ErrControlSkip
	}

	d.fields = fields
	return nil
}

func (d *persistenceDataControl) Handle(ctx context.Context, c reflect.Value, methodFunc func()) {
	elem := c.Elem() // controller should always be a pointer at this state
	for _, f := range d.fields {
		f.sendTo(elem)
	}
}

// PersistenceDataControl loads and re-stores
// the persistence data by scanning the original
// `TController.Value` instance of the user's controller.
func PersistenceDataControl() TControl {
	return &persistenceDataControl{}
}
