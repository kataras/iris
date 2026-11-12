package reflex

import (
	"reflect"
	"slices"
	"strings"
)

// IsStruct reports whether the "v" is a type of struct.
func IsStruct(v any) bool {
	if v == nil {
		return false
	}

	if typ, ok := v.(reflect.Type); ok { // it's actually a reflect.Type.
		return typ.Kind() == reflect.Struct
	}

	value, ok := v.(reflect.Value)
	if !ok {
		value = reflect.ValueOf(v)
	}

	return value.Kind() == reflect.Struct
}

// IsStructPtr reports whether the "v" is a type of *struct.
func IsStructPtr(v any) bool {
	if v == nil {
		return false
	}

	if typ, ok := v.(reflect.Type); ok { // it's actually a reflect.Type.
		return typ.Kind() == reflect.Pointer && typ.Elem().Kind() == reflect.Struct
	}

	value, ok := v.(reflect.Value)
	if !ok {
		value = reflect.ValueOf(v)
	}

	return value.Kind() == reflect.Pointer && value.Elem().Kind() == reflect.Struct
}

// LookupFields returns a slice of all fields containing a struct field
// of the given "fieldTag" of the "typ" struct. The fields returned
// are flatted and reclusive over fields with value of struct.
// Panics if "typ" is not a type of Struct.
func LookupFields(typ reflect.Type, fieldTag string) []reflect.StructField {
	opts := &LookupFieldsOptions{
		SkipUnexported:  true,
		SkipStructTypes: []reflect.Type{TimeType},
		FieldTag:        fieldTag,
	}
	fields, _ := LookupFieldsWithOptions(opts, typ) // no SkipField option which can resolve to an error is passed, so the error here is always nil.
	return fields
}

// LookupFieldsWithOptions returns a slice of all fields containing a struct field
// of the "typ" struct. The fields returned are flatted and reclusive over fields with value of struct.
// Panics if "typ" is not a type of Struct.
func LookupFieldsWithOptions(opts *LookupFieldsOptions, typ reflect.Type) ([]reflect.StructField, error) {
	fields, err := lookupFields(opts, typ, nil, 0, nil)
	if err != nil {
		return nil, err
	}
	return fields[0:len(fields):len(fields)], nil
}

// LookupFieldsOptions is the options for the LookupFields function.
type LookupFieldsOptions struct {
	// SkipUnexported reports whether to skip unexported fields.
	SkipUnexported bool
	// SkipTypes reports whether to skip fields that their type is in the list.
	SkipTypes []reflect.Type
	// SkipStructTypes reports whether to skip fields that their type is a struct and match the list.
	SkipStructTypes []reflect.Type
	// SkipEmbedded reports whether to skip embedded fields.
	SkipEmbedded bool
	// SkipPointers reports whether to skip pointer fields.
	SkipPointers bool
	// FieldTag is the tag name that the fields should contain.
	FieldTag string
	// SkipField reports whether to skip the field. If it returns true, the field will be skipped.
	SkipField func(field reflect.StructField) (bool, error)
	// FlatOffsets reports whether the caller intends to reach the returned fields
	// by pointer arithmetic (unsafe.Add) from the outermost struct value, using
	// the reported Offset.
	//
	// When true, a struct field held by POINTER is not flattened: its inner
	// fields live at a different allocation, so no offset relative to the outer
	// struct can reach them, and writing at such an offset would corrupt whatever
	// the outer struct actually holds there. The pointer field is reported as a
	// leaf instead, which lets the caller's field setter reject it cleanly.
	//
	// Struct fields held by value are flattened either way, and their Offset is
	// always reported relative to the outermost struct.
	FlatOffsets bool
}

// CanProccess reports whether the given struct field should be processed and,
// if so, returns its indirect type.
func (opts *LookupFieldsOptions) CanProccess(field reflect.StructField) (reflect.Type, bool, error) {
	return opts.canProccess(field, nil)
}

// canProccess is CanProccess with the chain of struct types already being
// walked, so a self-referential type does not recurse forever.
func (opts *LookupFieldsOptions) canProccess(field reflect.StructField, ancestors []reflect.Type) (reflect.Type, bool, error) {
	if opts.SkipUnexported && field.PkgPath != "" {
		return nil, false, nil
	}

	if opts.SkipEmbedded && field.Anonymous {
		return nil, false, nil
	}

	if slices.Contains(opts.SkipTypes, field.Type) {
		return nil, false, nil
	}

	if opts.SkipPointers && field.Type.Kind() == reflect.Pointer {
		return nil, false, nil
	}

	isChildStruct := false // NOTE: think of isEmbeddedStruct instead of isChildStruct, but this will require a change in the code below to take in mind the field.Anonymous boolean field.
	fieldIndirectType := IndirectType(field.Type)
	if fieldIndirectType.Kind() == reflect.Struct { // It's a struct inside a struct and it's in the list, skip it.
		isChildStruct = true
		if len(opts.SkipStructTypes) > 0 {
			if slices.Contains(opts.SkipStructTypes, fieldIndirectType) {
				return nil, false, nil
			}
		}
	}

	if tag := opts.FieldTag; tag != "" {
		tagValue := field.Tag.Get(tag)

		if isChildStruct {
			// If it's an (embedded) struct, no need for tag to be present, we only check for its omit option "-".

			if tagValue == "-" {
				// Skip embedded fields that has '-'.
				return nil, false, nil
			}

			// BUT
			// If there is an embedded inside an embedded without the param tag, the param tag check will be passed, must fix it:
			//
			// Note(@kataras): the ancestors check keeps a self-referential type
			// (type Node struct{ Next *Node }) from recursing here until the
			// goroutine stack overflows, which is a fatal runtime throw rather
			// than a recoverable panic. A type already on the chain was accepted
			// once, so accepting it again is the consistent answer.
			if !slices.Contains(ancestors, fieldIndirectType) {
				next := append(ancestors, fieldIndirectType)
				for embeddedStructField := range fieldIndirectType.Fields() {
					if _, canProcessEmbedded, _ := opts.canProccess(embeddedStructField, next); !canProcessEmbedded {
						return nil, false, nil
					}
				}
			}
		} else {
			if tagValue == "" || tagValue == "-" {
				// Skip fields that don't contain the 'fieldTag' tag or has '-'.
				return nil, false, nil
			}
		}
	}

	if opts.SkipField != nil {
		skip, err := opts.SkipField(field)
		if skip || err != nil {
			return nil, false, err
		}
	}

	return fieldIndirectType, true, nil
}

// lookupFields walks "typ" and returns its fields flattened over nested structs.
//
// "parentOffset" is the byte offset of "typ" inside the outermost struct, and
// "ancestors" is the chain of struct types currently being walked.
func lookupFields(opts *LookupFieldsOptions, typ reflect.Type, parentIndex []int, parentOffset uintptr, ancestors []reflect.Type) ([]reflect.StructField, error) {
	// Note(@kataras): a struct that reaches itself, directly or through another
	// type (type Node struct{ Next *Node }, or any tree/graph shaped DTO), would
	// otherwise recurse until the goroutine stack overflows - which is a fatal,
	// unrecoverable runtime throw, not a panic a handler can absorb. A repeated
	// type contributes no field tags that the first visit did not already report,
	// so stopping here loses nothing.
	if slices.Contains(ancestors, typ) {
		return nil, nil
	}
	ancestors = append(ancestors, typ)

	n := typ.NumField()
	fields := make([]reflect.StructField, 0, n)
	for i := range n {
		field := typ.Field(i)
		fieldType, ok, err := opts.canProccess(field, ancestors)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		// It's a struct inside a struct, flat it.
		//
		// When the caller reaches fields by offset (FlatOffsets), a struct held by
		// pointer must NOT be flattened: parentOffset+Offset would point somewhere
		// inside the outer struct, not into the pointed-to allocation.
		if fieldType.Kind() == reflect.Struct && (!opts.FlatOffsets || field.Type.Kind() == reflect.Struct) {
			structFields, err := lookupFields(opts, fieldType, append(parentIndex, i), parentOffset+field.Offset, ancestors)
			if err != nil {
				return nil, err
			}
			if nn := len(structFields); nn > 0 {
				fields = append(fields, structFields...)
				continue
			}
		}

		index := []int{i}
		if len(parentIndex) > 0 {
			index = append(parentIndex, i)
		}

		tmp := make([]int, len(index))
		copy(tmp, index)
		field.Index = tmp

		// Make the offset absolute, i.e. relative to the OUTERMOST struct.
		//
		// reflect reports Offset relative to the struct that DECLARES the field.
		// Because this function flattens nested and embedded structs into a single
		// list, a consumer doing unsafe.Add(outerPtr, field.Offset) would otherwise
		// write at the inner struct's offset applied to the outer struct - landing
		// on an unrelated field, or past the end of the allocation entirely.
		field.Offset += parentOffset

		fields = append(fields, field)
	}

	return fields, nil
}

type (
	// StructFieldsValuesOptions is the options for the StructFieldsValues function.
	StructFieldsValuesOptions struct {
		// SkipNonZero reports whether to skip non zero struct's fields values.
		SkipNonZero bool
	}

	// StructFieldValue is a struct field value with its index.
	// It's used in the StructFieldsValues function.
	StructFieldValue struct {
		Field reflect.StructField
		Value reflect.Value
	}
)

// StructFieldsValues returns a slice of all fields values of the "v" struct, with their indexes.
func StructFieldsValues(opts *StructFieldsValuesOptions, v reflect.Value, fields []reflect.StructField) []StructFieldValue {
	// values := make([]reflect.Value, 0, len(fields))
	// indexes := make([][]int, 0, len(fields))
	structFieldValues := make([]StructFieldValue, 0, len(fields))
	v = IndirectValue(v)
	// Check if v is valid and not nil to prevent nil pointer dereference
	// if !v.IsValid() || (v.Kind() == reflect.Pointer && v.IsNil()) {
	// 	v = reflect.New(v.Type()).Elem()
	// }

	for _, field := range fields {
		// if field.Type.Kind() == reflect.Pointer {
		// 	fmt.Printf("field: %#+v\n", field)
		// }
		value := v.FieldByIndex(field.Index)
		// if !value.IsValid() && value.Kind() == reflect.Pointer {
		// 	value = reflect.New(field.Type).Elem()
		// }
		if opts.SkipNonZero && !IsZero(value) {
			continue
		}

		structFieldValues = append(structFieldValues, StructFieldValue{
			Field: field,
			Value: value,
		})
	}

	return structFieldValues
}

// StructFieldsCountByTag returns the count of the fields that contains the "fieldTag" tag.
func StructFieldsCountByTag(fields []reflect.StructField, fieldTag string) int {
	count := 0
	for _, field := range fields {
		if field.Tag.Get(fieldTag) != "" {
			count++
		}
	}
	return count
}

// StructFieldsImplementers returns a slice of all fields that their type implements the "iface" explicitly.
func StructFieldsImplementers(fields []reflect.StructField, iface reflect.Type) []reflect.StructField {
	result := make([]reflect.StructField, 0, len(fields))
	for _, field := range fields {
		if field.Type.Implements(iface) {
			result = append(result, field)
		}
	}
	return result
}

// FirstStructFieldByTag returns the first field that contains the "fieldTag" tag.
func FirstStructFieldByTag(fields []reflect.StructField, fieldTag string) (reflect.StructField, bool) {
	for _, field := range fields {
		if field.Tag.Get(fieldTag) != "" {
			return field, true
		}
	}
	return reflect.StructField{}, false
}

// LookupUnderlineValueType returns the underline type of "v".
func LookupUnderlineValueType(v reflect.Value) (reflect.Value, reflect.Type) {
	typ := v.Type()
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		v = reflect.New(typ).Elem()
	}

	return v, typ
}

// NameOf returns the package name + the struct type's name.
func NameOf(v any) string {
	if v == nil {
		return ""
	}

	typ, ok := v.(reflect.Type)
	if !ok {
		if val, ok := v.(reflect.Value); ok {
			typ = val.Type()
		} else {
			typ = reflect.TypeOf(v)
		}
	}

	elemTyp := IndirectType(typ)

	typName := elemTyp.Name()

	// After generics feature this may return the type in [T] format, let's remove all these.
	for { // if found, remove it.
		start := strings.Index(typName, "[")
		if start < 1 {
			break
		}

		end := strings.Index(typName, "]") // if found, remove it.
		if end > start && len(typName) > end {
			typName = typName[:start] + typName[end+1:]
		} else {
			break
		}
	}

	pkgPath := elemTyp.PkgPath()
	if _, after, ok := strings.CutLast(pkgPath, "/"); ok {
		pkgPath = after
	}
	fullname := pkgPath + "." + typName

	return fullname
}

// PathParamReplacer is a replacer that replaces the characters in a path parameter name.
var PathParamReplacer = strings.NewReplacer("/", "_", "{", "_", "}", "_", ":", "_", "(", "_", ")", "_")

// ShortNameOf returns the short name of a struct type.
func ShortNameOf(v any) string {
	paramName := SnakeCase(NameOf(v))
	paramName = PathParamReplacer.Replace(paramName)

	// remove the first package's name.
	if firstDot := strings.Index(paramName, "."); firstDot > 0 {
		paramName = paramName[firstDot:]
	}
	paramName = strings.TrimLeftFunc(paramName, func(r rune) bool {
		return r == '.' || r == '_'
	})

	return paramName
}

// DeepClone recursively clones the fields of a struct.
func DeepClone[T any](t T) T {
	val := deepClone(ValueOf(t))
	return val.Interface().(T)
}

func deepClone(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Pointer:
		// Create a new pointer and clone the value it points to
		if v.IsNil() {
			return v
		}
		newPtr := reflect.New(v.Elem().Type())
		newPtr.Elem().Set(deepClone(v.Elem()))
		return newPtr
	case reflect.Struct:
		// Create a new struct and clone each field.
		newStruct := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			if newStruct.Field(i).Type().PkgPath() != "" {
				continue // it's non-exported.
			}
			newStruct.Field(i).Set(deepClone(v.Field(i)))
		}
		return newStruct
	case reflect.Slice:
		// Create a new slice and clone each element.
		newSlice := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		for i := 0; i < v.Len(); i++ {
			newSlice.Index(i).Set(deepClone(v.Index(i)))
		}
		return newSlice
	case reflect.Map:
		// Create a new map and clone each key-value pair.
		newMap := reflect.MakeMap(v.Type())
		for _, key := range v.MapKeys() {
			newMap.SetMapIndex(key, deepClone(v.MapIndex(key)))
		}
		return newMap
	default:
		// If it's not a pointer, struct, slice, or map, just return the value.
		return v
	}
}
