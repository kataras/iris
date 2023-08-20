package errors

import (
	"reflect"
	"strconv"
	"strings"
)

// ValidationError is an interface which IF
// it custom error types completes, then
// it can by mapped to a validation error.
//
// A validation error(s) can be given by ErrorCodeName's Validation or Err methods.
//
// Example can be found at:
//
//	https://github.com/kataras/iris/tree/main/_examples/routing/http-wire-errors/custom-validation-errors
type ValidationError interface {
	error

	GetField() string
	GetValue() interface{}
	GetReason() string
}

type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	var buf strings.Builder
	for i, err := range errs {
		buf.WriteByte('[')
		buf.WriteString(strconv.Itoa(i))
		buf.WriteByte(']')
		buf.WriteByte(' ')

		buf.WriteString(err.Error())

		if i < len(errs)-1 {
			buf.WriteByte(',')
			buf.WriteByte(' ')
		}
	}

	return buf.String()
}

// ValidationErrorMapper is the interface which
// custom validation error mappers should complete.
type ValidationErrorMapper interface {
	// The implementation must check the given "err"
	// and make decision if it's an error of validation
	// and if so it should return the value (err or another one)
	// and true as the last output argument.
	//
	// Outputs:
	//  1. the validation error(s) value
	//  2. true if the interface{} is an array, otherise false
	//  3. true if it's a validation error or false if not.
	MapValidationErrors(err error) (interface{}, bool, bool)
}

// ValidationErrorMapperFunc is an "ValidationErrorMapper" but in type of a function.
type ValidationErrorMapperFunc func(err error) (interface{}, bool, bool)

// MapValidationErrors completes the "ValidationErrorMapper" interface.
func (v ValidationErrorMapperFunc) MapValidationErrors(err error) (interface{}, bool, bool) {
	return v(err)
}

// read-only at serve time, holds the validation error mappers.
var validationErrorMappers []ValidationErrorMapper = []ValidationErrorMapper{
	ValidationErrorMapperFunc(func(err error) (interface{}, bool, bool) {
		switch e := err.(type) {
		case ValidationError:
			return e, false, true
		case ValidationErrors:
			return e, true, true
		default:
			return nil, false, false
		}
	}),
}

// RegisterValidationErrorMapper registers a custom
// implementation of validation error mapping.
// Call it on program initilization, main() or init() functions.
func RegisterValidationErrorMapper(m ValidationErrorMapper) {
	validationErrorMappers = append(validationErrorMappers, m)
}

// RegisterValidationErrorMapperFunc registers a custom
// function implementation of validation error mapping.
// Call it on program initilization, main() or init() functions.
func RegisterValidationErrorMapperFunc(fn func(err error) (interface{}, bool, bool)) {
	validationErrorMappers = append(validationErrorMappers, ValidationErrorMapperFunc(fn))
}

type validationErrorTypeMapper struct {
	types []reflect.Type
}

var _ ValidationErrorMapper = (*validationErrorTypeMapper)(nil)

func (v *validationErrorTypeMapper) MapValidationErrors(err error) (interface{}, bool, bool) {
	errType := reflect.TypeOf(err)
	for _, typ := range v.types {
		if equalTypes(errType, typ) {
			return err, false, true
		}

		// a slice is given but the underline type is registered.
		if errType.Kind() == reflect.Slice {
			if equalTypes(errType.Elem(), typ) {
				return err, true, true
			}
		}
	}

	return nil, false, false
}

func equalTypes(err reflect.Type, binding reflect.Type) bool {
	return err == binding
	// return binding.AssignableTo(err)
}

// NewValidationErrorTypeMapper returns a validation error mapper
// which compares the error with one or more of the given "types",
// through reflection. Each of the given types MUST complete the
// standard error type, so it can be passed through the error code.
func NewValidationErrorTypeMapper(types ...error) ValidationErrorMapper {
	typs := make([]reflect.Type, 0, len(types))
	for _, typ := range types {
		v, ok := typ.(reflect.Type)
		if !ok {
			v = reflect.TypeOf(typ)
		}

		typs = append(typs, v)
	}

	return &validationErrorTypeMapper{
		types: typs,
	}
}

// AsValidationErrors reports wheether the given "err" is a type of validation error(s).
// Its behavior can be modified before serve-time
// through the "RegisterValidationErrorMapper" function.
func AsValidationErrors(err error) (interface{}, bool) {
	if err == nil {
		return nil, false
	}

	for _, m := range validationErrorMappers {
		if errs, isArray, ok := m.MapValidationErrors(err); ok {
			if !isArray { // ensure always-array on Validation field of the http error.
				return []interface{}{errs}, true
			}
			return errs, true
		}
	}

	return nil, false
}
