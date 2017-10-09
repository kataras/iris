package field

import (
	"reflect"
)

// Field is a controller's field
// contains all the necessary, internal, information
// to work with.
type Field struct {
	Name string // the field's original name
	// but if a tag with `name: "other"`
	// exist then this fill is filled, otherwise it's the same as the Name.
	TagName string
	Index   int
	Type    reflect.Type
	Value   reflect.Value

	embedded *Field
}

// GetIndex returns all the "dimensions"
// of the controller struct field's position that this field is referring to,
// recursively.
// Usage: elem.FieldByIndex(field.getIndex())
// for example the {0,1} means that the field is on the second field of the first's
// field of this struct.
func (ff Field) GetIndex() []int {
	deepIndex := []int{ff.Index}

	if emb := ff.embedded; emb != nil {
		deepIndex = append(deepIndex, emb.GetIndex()...)
	}

	return deepIndex
}

// GetType returns the type of the referring field, recursively.
func (ff Field) GetType() reflect.Type {
	typ := ff.Type
	if emb := ff.embedded; emb != nil {
		return emb.GetType()
	}

	return typ
}

// GetFullName returns the full name of that field
// i.e: UserController.SessionController.Manager,
// it's useful for debugging only.
func (ff Field) GetFullName() string {
	name := ff.Name

	if emb := ff.embedded; emb != nil {
		return name + "." + emb.GetFullName()
	}

	return name
}

// GetTagName returns the tag name of the referring field
// recursively.
func (ff Field) GetTagName() string {
	name := ff.TagName

	if emb := ff.embedded; emb != nil {
		return emb.GetTagName()
	}

	return name
}

// checkVal checks if that value
// is valid to be set-ed to the new controller's instance.
// Used on binder.
func checkVal(val reflect.Value) bool {
	return val.IsValid() && (val.Kind() == reflect.Ptr && !val.IsNil()) && val.CanInterface()
}

// GetValue returns a valid value of the referring field, recursively.
func (ff Field) GetValue() interface{} {
	if ff.embedded != nil {
		return ff.embedded.GetValue()
	}

	if checkVal(ff.Value) {
		return ff.Value.Interface()
	}

	return "undefinied value"
}

// SendTo should be used when this field or its embedded
// has a Value on it.
// It sets the field's value to the "elem" instance, it's the new controller.
func (ff Field) SendTo(elem reflect.Value) {
	// note:
	// we don't use the getters here
	// because we do recursively search by our own here
	// to be easier to debug if ever needed.
	if embedded := ff.embedded; embedded != nil {
		if ff.Index >= 0 {
			embedded.SendTo(elem.Field(ff.Index))
		}
		return
	}
	elemField := elem.Field(ff.Index)
	if elemField.Kind() == reflect.Ptr && !elemField.IsNil() {
		return
	}

	elemField.Set(ff.Value)
}

// lookupTagName checks if the "elemField" struct's field
// contains a tag `name`, if it contains then it returns its value
// otherwise returns the field's original Name.
func lookupTagName(elemField reflect.StructField) string {
	vname := elemField.Name

	if taggedName, ok := elemField.Tag.Lookup("name"); ok {
		vname = taggedName
	}
	return vname
}

// LookupFields iterates all "elem"'s fields and its fields
// if structs, recursively.
// Compares them to the "matcher", if they passed
// then it executes the "handler" if any,
// the handler can change the field as it wants to.
//
// It finally returns that collection of the valid fields, can be empty.
func LookupFields(elem reflect.Type, matcher func(reflect.StructField) bool, handler func(*Field)) (fields []Field) {
	for i, n := 0, elem.NumField(); i < n; i++ {
		elemField := elem.Field(i)
		if matcher(elemField) {
			field := Field{
				Index:   i,
				Name:    elemField.Name,
				TagName: lookupTagName(elemField),
				Type:    elemField.Type,
			}

			if handler != nil {
				handler(&field)
			}

			// we area inside the correct type
			fields = append(fields, field)
			continue
		}

		f := lookupStructField(elemField.Type, matcher, handler)
		if f != nil {
			fields = append(fields, Field{
				Index:    i,
				Name:     elemField.Name,
				Type:     elemField.Type,
				embedded: f,
			})
		}

	}
	return
}

// lookupStructField is here to search for embedded field only,
// is working with the "lookupFields".
// We could just one one function
// for both structured (embedded) fields and normal fields
// but we keep that as it's, a new function like this
// is easier for debugging, if ever needed.
func lookupStructField(elem reflect.Type, matcher func(reflect.StructField) bool, handler func(*Field)) *Field {
	// fmt.Printf("lookup struct for elem: %s\n", elem.Name())
	// ignore if that field is not a struct
	if elem.Kind() != reflect.Struct {
		return nil
	}

	// search by fields.
	for i, n := 0, elem.NumField(); i < n; i++ {
		elemField := elem.Field(i)

		if matcher(elemField) {
			// we area inside the correct type.
			f := &Field{
				Index:   i,
				Name:    elemField.Name,
				TagName: lookupTagName(elemField),
				Type:    elemField.Type,
			}

			if handler != nil {
				handler(f)
			}

			return f
		}

		// if field is struct and the value is struct
		// then try inside its fields for a compatible
		// field type.
		if elemField.Type.Kind() == reflect.Struct { // 3-level
			elemFieldEmb := elem.Field(i)
			f := lookupStructField(elemFieldEmb.Type, matcher, handler)
			if f != nil {
				fp := &Field{
					Index:    i,
					Name:     elemFieldEmb.Name,
					TagName:  lookupTagName(elemFieldEmb),
					Type:     elemFieldEmb.Type,
					embedded: f,
				}
				return fp
			}
		}
	}
	return nil
}
