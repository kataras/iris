package validation

import (
	"fmt"
	"strings"
)

// StringError describes a string field validation error.
type StringError struct{ *FieldError[string] }

// String returns a new string validation error.
func String(field string, value string) *StringError {
	return &StringError{Field(field, value)}
}

// NotEmpty adds an error if the string is empty.
func (e *StringError) NotEmpty() *StringError {
	e.Func(NotEmpty)
	return e
}

// Fullname adds an error if the string is not a full name.
func (e *StringError) Fullname() *StringError {
	e.Func(Fullname)
	return e
}

// Length adds an error if the string length is not in the given range.
func (e *StringError) Length(min, max int) *StringError {
	e.Func(StringLength(min, max))
	return e
}

// NotEmpty accepts any string and returns a message if the value is empty.
func NotEmpty(s string) string {
	if s == "" {
		return "must not be empty"
	}

	return ""
}

// Fullname accepts any string and returns a message if the value is not a full name.
func Fullname(s string) string {
	if len(strings.Split(s, " ")) < 2 {
		return "must contain first and last name"
	}

	return ""
}

// StringLength accepts any string and returns a message if the length is not in the given range.
func StringLength(min, max int) func(s string) string {
	return func(s string) string {
		n := len(s)

		if min == max {
			if n != min {
				return fmt.Sprintf("must be %d characters", min)
			}
		}

		if n < min || n > max {
			return fmt.Sprintf("must be between %d and %d characters", min, max)
		}

		return ""
	}
}
