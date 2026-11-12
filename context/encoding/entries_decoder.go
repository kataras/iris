package encoding

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/kataras/iris/v14/core/pkg/reflex"

	"uuid"
)

// FieldSetter is a function that sets a field value.
// It's used by the decoder to set the field value.
// The field is the reflect.StructField of the field.
// The ptr is a pointer to the field.
// The value is the value to set.
type FieldSetter[V any] func(field *reflect.StructField, ptr unsafe.Pointer, value V) error

// EntriesDecoderOptions is the options for the path params decoder.
// See `NewEntriesDecoder` package-level function for more.
type EntriesDecoderOptions[V any] struct {
	// FieldTag is the struct tag to look.
	FieldTag string
	// Strict is a flag that determines the behavior of the decoder when it encounters an unknown path param.
	// If true then it should not have any extra or missing fields. Marks all fields as required.
	// Defaults to false.
	Strict bool
	// FieldSetter is the field setter function.
	// Defaults to DefaultPathFieldSetter.
	FieldSetter FieldSetter[V]
}

func (opts EntriesDecoderOptions[V]) get(defaultTag string, defaultSetter FieldSetter[V]) EntriesDecoderOptions[V] {
	options := opts
	if options.FieldTag == "" {
		options.FieldTag = defaultTag
	}

	if options.FieldSetter == nil { // both context and path params have the same exact field setter and decoder except their struct field tag name.
		options.FieldSetter = defaultSetter
	}

	return options
}

type (
	// EntriesDecoder decodes the context values or path params to the target struct value of T.
	// Create one through the `NewEntriesDecoder` package-level function and use its
	// `DecodeEntries` method.
	EntriesDecoder[T, V any] struct {
		options                   EntriesDecoderOptions[V]
		typ                       reflect.Type // elem type of struct.
		implementsStructValidator bool
		fields                    map[string]*pathFieldInfo[V]
		hasRequired               bool // if true then at least one field is required. See fieldInfo.required too.
	}

	pathFieldInfo[V any] struct {
		field  *reflect.StructField
		offset uintptr
		setter FieldSetter[V]

		required bool // if true then it should be found in the context values or path params. Defaults to false.
	}
)

// NewEntriesDecoder returns a new entries decoder.
// It returns an error if T is not a struct type, the decoder forms the *T target by itself.
// The options can be empty.
// See `EntriesDecoderOptions` struct and its `DecodeEntries` method for more.
func NewEntriesDecoder[T, V any](opts EntriesDecoderOptions[V]) (*EntriesDecoder[T, V], error) {
	typ, implementsStructValidator, err := getStructElem((*T)(nil))
	if err != nil {
		return nil, err
	}

	d := &EntriesDecoder[T, V]{
		options:                   opts,
		typ:                       typ, // keep a reference of the reflect.Type for any future use.
		implementsStructValidator: implementsStructValidator,
		fields:                    make(map[string]*pathFieldInfo[V]),
	}
	d.makeFields()

	if d.hasRequired && implementsStructValidator {
		return nil, fmt.Errorf("struct %s implements StructValidator and has required fields, use either the `Validate` method or the 'required' field tag to validate the struct", typ.String())
	}

	return d, nil
}

func (d *EntriesDecoder[T, V]) makeFields() {
	typ := d.typ
	opts := d.options

	// for i := 0; i < typ.NumField(); i++ {
	// 	field := typ.Field(i)
	// 	if field.PkgPath != "" {
	// 		continue // Skip unexported fields.
	// 	}
	// paramTag, ok := field.Tag.Lookup(opts.FieldTag)
	// if !ok {
	// 	continue // Skip fields without the given "fieldTag" tag.
	// }

	fieldOpts := &reflex.LookupFieldsOptions{
		SkipUnexported:  true,
		SkipStructTypes: []reflect.Type{reflex.TimeType},
		// SkipPointers:    true,
		FieldTag: opts.FieldTag,
		// This decoder reaches every field by unsafe.Add(targetPtr, info.offset),
		// so it needs offsets relative to the outermost struct and it must not be
		// handed fields that live behind a pointer. See LookupFieldsOptions.
		FlatOffsets: true,
	}
	fields, _ := reflex.LookupFieldsWithOptions(fieldOpts, typ)
	for _, field := range fields {
		paramTag := field.Tag.Get(opts.FieldTag)
		paramOptions := strings.Split(paramTag, ",")
		paramName := paramOptions[0]
		required := opts.Strict
		if len(paramOptions) > 1 {
			for _, opt := range paramOptions[1:] {
				switch opt {
				case "required":
					required = true || required // always true if strict.
				default:
					// ignore unknown options.
				}
			}
		}

		d.fields[paramName] = &pathFieldInfo[V]{
			field:    &field,
			offset:   field.Offset,
			setter:   opts.FieldSetter,
			required: required,
		}

		if required {
			d.hasRequired = true
		}
	}
}

// HasRequired reports whether the decoder has at least one required field.
func (d *EntriesDecoder[T, V]) HasRequired() bool {
	return d.hasRequired
}

func getStructElem(v any) (reflect.Type, bool, error) {
	typ := reflex.TypeOf(v)
	if typ.Kind() != reflect.Pointer {
		return nil, false, fmt.Errorf("expected a pointer to struct, got %s", typ.String())
	}
	implementsStructValidator := isStructValidator(typ)

	typ = typ.Elem()
	if typ.Kind() != reflect.Struct {
		return nil, false, fmt.Errorf("expected a pointer to struct, got %s", typ.String())
	}

	return typ, implementsStructValidator, nil
}

// StructValidator is the interface that wraps the Validate method.
// Validate validates the struct value.
// It returns an error if the struct value is invalid.
// The error can be a *ValidationError.
//
// Use this interface to validate the struct value after decoding the context values or path params or query or header or form data.
// The struct value must implement this interface to be validated, otherwise, the decoder will fallback to the "required" struct field option.
type StructValidator interface {
	Validate() error
}

func isStructValidator(typ reflect.Type) bool {
	return typ.Implements(reflect.TypeFor[StructValidator]())
}

func (d *EntriesDecoder[T, V]) validateTarget(target any) error {
	if d.implementsStructValidator {
		if sv, ok := target.(StructValidator); ok {
			if err := sv.Validate(); err != nil {
				if _, isAlreadyValidatiorErr := err.(*ValidationError); !isAlreadyValidatiorErr {
					return newFieldError("", nil, err.Error()) // prints just the reason as validation error with rest/errors.
				}
				return err
			}
		}
	}

	return nil
}

type validationReason string

const (
	validationReasonRequired validationReason = "required field"
	validationReasonExtra    validationReason = "extra field"
)

func (r validationReason) String() string {
	return string(r)
}

type ValidationError struct {
	Field  string `json:"field,omitempty"`
	Value  any    `json:"value,omitempty"`
	Reason string `json:"reason"`
}

func newFieldError(field string, value any, reason string) *ValidationError {
	return &ValidationError{
		Field:  field,
		Value:  value,
		Reason: reason,
	}
}

func newRequiredFieldError(field string, value any) *ValidationError {
	return newFieldError(field, value, validationReasonRequired.String())
}

func newExtraFieldError(field string, value any) *ValidationError {
	return newFieldError(field, value, validationReasonExtra.String())
}

// IsRequired reports whether the error is a required field error.
func (e *ValidationError) IsRequired() bool {
	return e.Reason == validationReasonRequired.String()
}

// IsExtra reports whether the error is an extra field error.
func (e *ValidationError) IsExtra() bool {
	return e.Reason == validationReasonExtra.String()
}

// Error implements the standard error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("field: %s, value: %v, reason: %s", e.Field, e.Value, e.Reason)
}

// GetReason implements the rest/validation error interface.
func (e *ValidationError) GetReason() string {
	return string(e.Reason)
}

// DecodeEntries decodes the context values or path params to the target struct value of T.
func (d *EntriesDecoder[T, V]) DecodeEntries(store Entries[V], target *T) error {
	targetPtr := unsafe.Pointer(target)

	if d.hasRequired { // all required or at least one. We loop over the struct's fields instead of the params here.
		if d.options.Strict {
			if expectedFieldsLen, paramsLen := len(d.fields), len(store); expectedFieldsLen != paramsLen {
				if paramsLen > expectedFieldsLen {
					// extract the extra params.
					extraFields := make([]string, 0, paramsLen-expectedFieldsLen)
					extraFieldsValues := make([]string, 0, paramsLen-expectedFieldsLen)
					for _, param := range store {
						if _, ok := d.fields[param.Key]; !ok {
							extraFields = append(extraFields, param.Key)
							extraFieldsValues = append(extraFieldsValues, param.String())
						}
					}

					return newExtraFieldError(strings.Join(extraFields, ", "), strings.Join(extraFieldsValues, ", "))
				}
			}
		}

		for paramName, info := range d.fields {
			value, ok := store.Lookup(paramName) // slower path.
			if !ok {
				if info.required {
					return newRequiredFieldError(paramName, "")
				}

				continue
			}

			fieldPtr := unsafe.Add(targetPtr, info.offset)
			if err := info.setter(info.field, fieldPtr, value); err != nil {
				return fmt.Errorf("%s->%s: %w", paramName, info.field.Name, err)
			}
		}

		if err := d.validateTarget(target); err != nil {
			return err
		}

		return nil
	}

	for _, entry := range store { // fast path.
		key := entry.Key
		value := entry.Value // here we will modify it to accept the typed one, as this happens after macro evaluator.
		if info, ok := d.fields[key]; ok {
			// if info.kind == reflect.Pointer {
			// 	newPtr := reflect.New(info.field.Type.Elem())
			// 	d.DecodeEntries(store, newPtr.Interface())
			// 	continue
			// } No, don't support pointers here, as we must know the parent type and what params they match, and if we give the option for embedded pointers, there are might conflicts in param names used there.

			fieldPtr := unsafe.Add(targetPtr, info.offset)
			if err := info.setter(info.field, fieldPtr, value); err != nil {
				return fmt.Errorf("%s->%s: %w", key, info.field.Name, err)
			}
			// Doesn't work:
			// *(*any)(fieldPtr) = *(*any)(unsafe.Pointer(&value))
		} else if d.options.Strict {
			return newExtraFieldError(key, value)
		}
	}

	if err := d.validateTarget(target); err != nil {
		return err
	}

	return nil
}

var ErrUnsupportedType = fmt.Errorf("unsupported type")

// ErrTypeMismatch is returned when a decoded value's type does not match
// the destination struct field's type.
var ErrTypeMismatch = fmt.Errorf("type mismatch")

// DefaultPathFieldSetter is the default field setter for context values or path params.
//
// Note(@kataras): every branch dispatches on the FIELD's declared type, never on
// the value's runtime type, because "ptr" points at the field and the write size
// is decided by the type we cast it to. Dispatching on the value would let a
// path parameter that the macro evaluator produced as a string (the type of an
// untyped "{id}") be written as a 16-byte string header over, say, an 8-byte
// int64 field - clobbering the next field with a client-controlled value.
func DefaultPathFieldSetter[V any](field *reflect.StructField, ptr unsafe.Pointer, value V) error {
	v := any(value)

	switch field.Type {
	case reflex.StringType:
		switch value := v.(type) {
		case string:
			*(*string)(ptr) = value
		case uuid.UUID:
			// The "uuid" macro stores a uuid.UUID; a string field receives
			// its canonical text.
			*(*string)(ptr) = value.String()
		default:
			return typeMismatch(field, v)
		}
	case reflex.IntType:
		n, err := signedValue(field, v, strconv.IntSize)
		if err != nil {
			return err
		}
		*(*int)(ptr) = int(n)
	case reflex.Int8Type:
		n, err := signedValue(field, v, 8)
		if err != nil {
			return err
		}
		*(*int8)(ptr) = int8(n)
	case reflex.Int16Type:
		n, err := signedValue(field, v, 16)
		if err != nil {
			return err
		}
		*(*int16)(ptr) = int16(n)
	case reflex.Int32Type:
		n, err := signedValue(field, v, 32)
		if err != nil {
			return err
		}
		*(*int32)(ptr) = int32(n)
	case reflex.Int64Type:
		n, err := signedValue(field, v, 64)
		if err != nil {
			return err
		}
		*(*int64)(ptr) = n
	case reflex.UintType:
		n, err := unsignedValue(field, v, strconv.IntSize)
		if err != nil {
			return err
		}
		*(*uint)(ptr) = uint(n)
	case reflex.Uint8Type:
		n, err := unsignedValue(field, v, 8)
		if err != nil {
			return err
		}
		*(*uint8)(ptr) = uint8(n)
	case reflex.Uint16Type:
		n, err := unsignedValue(field, v, 16)
		if err != nil {
			return err
		}
		*(*uint16)(ptr) = uint16(n)
	case reflex.Uint32Type:
		n, err := unsignedValue(field, v, 32)
		if err != nil {
			return err
		}
		*(*uint32)(ptr) = uint32(n)
	case reflex.Uint64Type:
		n, err := unsignedValue(field, v, 64)
		if err != nil {
			return err
		}
		*(*uint64)(ptr) = n
	case reflex.BoolType:
		b, ok := v.(bool)
		if !ok {
			return typeMismatch(field, v)
		}
		*(*bool)(ptr) = b
	case reflex.Float32Type:
		f, ok := floatValue(v)
		if !ok {
			return typeMismatch(field, v)
		}
		*(*float32)(ptr) = float32(f)
	case reflex.Float64Type:
		f, ok := floatValue(v)
		if !ok {
			return typeMismatch(field, v)
		}
		*(*float64)(ptr) = f
	case reflex.ArrayStringTyp:
		switch value := v.(type) {
		case []string:
			*(*[]string)(ptr) = value
		case string:
			// The param was a string but the field is a []string: split by "/",
			// which is what the ":path" param type produces.
			//
			// plus one allocation here, i can avoid it by making the macro evaluator
			// to return a slice of strings instead of a single string on the context value or path param,
			// but this will reduce its performance on macro's handler#Eval, so we stick with this Split call.
			*(*[]string)(ptr) = strings.Split(value, "/")
		default:
			return typeMismatch(field, v)
		}
	case reflex.ArrayIntegerTyp:
		n, ok := v.([]int)
		if !ok {
			return typeMismatch(field, v)
		}
		*(*[]int)(ptr) = n
	case reflex.TimeType:
		t, ok := v.(time.Time)
		if !ok {
			return typeMismatch(field, v)
		}
		*(*time.Time)(ptr) = t
	case reflex.WeekdayType:
		d, ok := v.(time.Weekday)
		if !ok {
			return typeMismatch(field, v)
		}
		*(*time.Weekday)(ptr) = d
	case reflex.UUIDType:
		// The "uuid" macro stores a uuid.UUID; an untyped "{id}" or a context
		// value may carry the text form.
		switch value := v.(type) {
		case uuid.UUID:
			*(*uuid.UUID)(ptr) = value
		case string:
			id, err := uuid.Parse(value)
			if err != nil {
				return typeMismatch(field, v)
			}
			*(*uuid.UUID)(ptr) = id
		default:
			return typeMismatch(field, v)
		}
	case reflex.DurationType:
		switch value := v.(type) {
		case time.Duration:
			*(*time.Duration)(ptr) = value
		case string:
			d, err := time.ParseDuration(value)
			if err != nil {
				return typeMismatch(field, v)
			}
			*(*time.Duration)(ptr) = d
		default:
			return typeMismatch(field, v)
		}
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedType, field.Type.String())
	}

	return nil
}

func typeMismatch(field *reflect.StructField, value any) error {
	return fmt.Errorf("%w: field %s is %s, got %T", ErrTypeMismatch, field.Name, field.Type.String(), value)
}

func overflows(field *reflect.StructField, value any) error {
	return fmt.Errorf("%w: field %s is %s, value %v is out of range", ErrTypeMismatch, field.Name, field.Type.String(), value)
}

// signedValue converts any numeric value the macro evaluator can produce into an
// int64 that fits in "bitSize" bits.
//
// The macro types and the destination field types are chosen independently by
// the end-developer ("{id:int}" bound to an int64 field is both reasonable and
// common), so widening and narrowing are supported - but the range is checked,
// because the caller writes exactly bitSize/8 bytes at the field's address.
func signedValue(field *reflect.StructField, value any, bitSize int) (int64, error) {
	var n int64

	switch v := value.(type) {
	case int:
		n = int64(v)
	case int8:
		n = int64(v)
	case int16:
		n = int64(v)
	case int32:
		n = int64(v)
	case int64:
		n = v
	case uint:
		if uint64(v) > math.MaxInt64 {
			return 0, overflows(field, value)
		}
		n = int64(v)
	case uint8:
		n = int64(v)
	case uint16:
		n = int64(v)
	case uint32:
		n = int64(v)
	case uint64:
		if v > math.MaxInt64 {
			return 0, overflows(field, value)
		}
		n = int64(v)
	default:
		return 0, typeMismatch(field, value)
	}

	if bitSize < 64 {
		limit := int64(1) << (bitSize - 1)
		if n < -limit || n > limit-1 {
			return 0, overflows(field, value)
		}
	}

	return n, nil
}

// unsignedValue is signedValue for unsigned destination fields.
// A negative value never fits, so it is reported rather than wrapped around.
func unsignedValue(field *reflect.StructField, value any, bitSize int) (uint64, error) {
	var n uint64

	switch v := value.(type) {
	case int:
		if v < 0 {
			return 0, overflows(field, value)
		}
		n = uint64(v)
	case int8:
		if v < 0 {
			return 0, overflows(field, value)
		}
		n = uint64(v)
	case int16:
		if v < 0 {
			return 0, overflows(field, value)
		}
		n = uint64(v)
	case int32:
		if v < 0 {
			return 0, overflows(field, value)
		}
		n = uint64(v)
	case int64:
		if v < 0 {
			return 0, overflows(field, value)
		}
		n = uint64(v)
	case uint:
		n = uint64(v)
	case uint8:
		n = uint64(v)
	case uint16:
		n = uint64(v)
	case uint32:
		n = uint64(v)
	case uint64:
		n = v
	default:
		return 0, typeMismatch(field, value)
	}

	if bitSize < 64 && n > (uint64(1)<<bitSize)-1 {
		return 0, overflows(field, value)
	}

	return n, nil
}

func floatValue(value any) (float64, bool) {
	switch v := value.(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}
