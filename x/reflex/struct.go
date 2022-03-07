package reflex

import "reflect"

// LookupFields returns a slice of all fields containing a struct field
// of the given "fieldTag" of the "typ" struct. The fields returned
// are flatted and reclusive over fields with value of struct.
// Panics if "typ" is not a type of Struct.
func LookupFields(typ reflect.Type, fieldTag string) []reflect.StructField {
	fields := lookupFields(typ, fieldTag, nil)
	return fields[0:len(fields):len(fields)]
}

func lookupFields(typ reflect.Type, fieldTag string, parentIndex []int) []reflect.StructField {
	n := typ.NumField()
	fields := make([]reflect.StructField, 0, n)
	checkTag := fieldTag != ""
	for i := 0; i < n; i++ {
		field := typ.Field(i)
		if field.PkgPath != "" { // skip unexported fields.
			continue
		}

		if checkTag {
			if v := field.Tag.Get(fieldTag); v == "" || v == "-" {
				// Skip fields that don't contain the 'fieldTag' tag or has '-'.
				continue
			}
		}

		fieldType := IndirectType(field.Type)

		if fieldType.Kind() == reflect.Struct { // It's a struct inside a struct and it's not time, flat it.
			if fieldType != TimeType {
				structFields := lookupFields(fieldType, fieldTag, append(parentIndex, i))
				if nn := len(structFields); nn > 0 {
					fields = append(fields, structFields...)
					continue
				}
			}
		}

		index := []int{i}
		if len(parentIndex) > 0 {
			index = append(parentIndex, i)
		}

		tmp := make([]int, len(index))
		copy(tmp, index)
		field.Index = tmp

		fields = append(fields, field)
	}

	return fields
}

// LookupUnderlineValueType returns the underline type of "v".
func LookupUnderlineValueType(v reflect.Value) (reflect.Value, reflect.Type) {
	typ := v.Type()
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		v = reflect.New(typ).Elem()
	}

	return v, typ
}
