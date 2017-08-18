package activator

import (
	"reflect"

	"github.com/kataras/iris/context"
)

type field struct {
	Name  string // by-defaultis the field's name but if `name: "other"` then it's overridden.
	Index int
	Type  reflect.Type
	Value reflect.Value

	embedded *field
}

func (ff field) sendTo(elem reflect.Value) {
	if embedded := ff.embedded; embedded != nil {
		if ff.Index >= 0 {
			embedded.sendTo(elem.Field(ff.Index))
		}
		return
	}
	elemField := elem.Field(ff.Index)
	if elemField.Kind() == reflect.Ptr && !elemField.IsNil() {
		return
	}

	elemField.Set(ff.Value)
}

func lookupFields(t *TController, validator func(reflect.StructField) bool) (fields []field) {
	elem := t.Type.Elem()

	for i, n := 0, elem.NumField(); i < n; i++ {
		elemField := elem.Field(i)
		valF := t.Value.Field(i)

		// catch persistence data by tags, i.e:
		// MyData string `iris:"persistence"`
		if validator(elemField) {
			name := elemField.Name
			if nameTag, ok := elemField.Tag.Lookup("name"); ok {
				name = nameTag
			}

			f := field{
				Name:  name,
				Index: i,
				Type:  elemField.Type,
			}

			if valF.IsValid() || (valF.Kind() == reflect.Ptr && !valF.IsNil()) {
				val := reflect.ValueOf(valF.Interface())
				if val.IsValid() || (val.Kind() == reflect.Ptr && !val.IsNil()) {
					f.Value = val
				}
			}

			fields = append(fields, f)
		}
	}
	return
}

type persistenceDataControl struct {
	fields []field
}

func (d *persistenceDataControl) Load(t *TController) error {
	fields := lookupFields(t, func(f reflect.StructField) bool {
		if tag, ok := f.Tag.Lookup("iris"); ok {
			if tag == "persistence" {
				return true
			}
		}
		return false
	})

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
