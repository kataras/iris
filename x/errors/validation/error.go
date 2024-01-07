package validation

import (
	"fmt"

	"github.com/kataras/iris/v12/x/errors"
)

// FieldError describes a field validation error.
// It completes the errors.ValidationError interface.
type FieldError[T any] struct {
	Field  string `json:"field"`
	Value  T      `json:"value"`
	Reason string `json:"reason"`
}

// Field returns a new validation error.
//
// Use its Func method to add validations over this field.
func Field[T any](field string, value T) *FieldError[T] {
	return &FieldError[T]{Field: field, Value: value}
}

// Error completes the standard error interface.
func (e *FieldError[T]) Error() string {
	return fmt.Sprintf("field %q got invalid value of %v: reason: %s", e.Field, e.Value, e.Reason)
}

// GetField returns the field name.
func (e *FieldError[T]) GetField() string {
	return e.Field
}

// GetValue returns the value of the field.
func (e *FieldError[T]) GetValue() any {
	return e.Value
}

// GetReason returns the reason of the validation error.
func (e *FieldError[T]) GetReason() string {
	return e.Reason
}

// IsZero reports whether the error is nil or has an empty reason.
func (e *FieldError[T]) IsZero() bool {
	return e == nil || e.Reason == ""
}

func (e *FieldError[T]) joinReason(reason string) {
	if reason == "" {
		return
	}

	if e.Reason == "" {
		e.Reason = reason
	} else {
		e.Reason += ", " + reason
	}
}

// Func accepts a variadic number of functions which accept the value of the field
// and return a string message if the value is invalid.
// It joins the reasons into one.
func (e *FieldError[T]) Func(fns ...func(value T) string) *FieldError[T] {
	for _, fn := range fns {
		e.joinReason(fn(e.Value))
	}

	return e
}

// Join joins the given validation errors into one.
func Join(errs ...errors.ValidationError) error { // note that here we return the standard error type instead of the errors.ValidationError in order to make the error nil instead of ValidationErrors(nil) on empty slice.
	if len(errs) == 0 {
		return nil
	}

	joinedErrs := make(errors.ValidationErrors, 0, len(errs))
	for _, err := range errs {
		if err == nil || err.GetReason() == "" {
			continue
		}

		joinedErrs = append(joinedErrs, err)
	}

	if len(joinedErrs) == 0 {
		return nil
	}

	return joinedErrs
}
