package errors

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// AsValidationErrors reports wheether the given "err" is a type of validation error(s).
func AsValidationErrors(err error) (ValidationErrors, bool) {
	if err == nil {
		return nil, false
	}

	switch e := err.(type) {
	case ValidationError:
		return ValidationErrors{e}, true
	case ValidationErrors:
		return e, true
	case *ValidationErrors:
		return *e, true
	default:
		return nil, false
	}
}

// ValueValidator is a generic interface which can be used to check if the value is valid for insert (or for comparison inside another validation step).
// Useful for enums.
// Should return a non-empty string on validation error, that string is the failure reason.
type ValueValidator interface {
	Validate() string
}

// ValidationError describes a field validation error.
type ValidationError struct {
	Field  string      `json:"field" yaml:"Field"`
	Value  interface{} `json:"value" yaml:"Value"`
	Reason string      `json:"reason" yaml:"Reason"`
}

// Error completes the standard error interface.
func (e ValidationError) Error() string {
	return sprintf("field %q got invalid value of %v: reason: %s", e.Field, e.Value, e.Reason)
}

// ValidationErrors is just a custom type of ValidationError slice.
type ValidationErrors []ValidationError

// Error completes the error interface.
func (e ValidationErrors) Error() string {
	var buf strings.Builder
	for i, err := range e {
		buf.WriteByte('[')
		buf.WriteString(strconv.Itoa(i))
		buf.WriteByte(']')
		buf.WriteByte(' ')

		buf.WriteString(err.Error())

		if i < len(e)-1 {
			buf.WriteByte(',')
			buf.WriteByte(' ')
		}
	}

	return buf.String()
}

// Is reports whether the given "err" is a type of validation error or validation errors.
func (e ValidationErrors) Is(err error) bool {
	if err == nil {
		return false
	}

	switch err.(type) {
	case ValidationError:
		return true
	case *ValidationError:
		return true
	case ValidationErrors:
		return true
	case *ValidationErrors:
		return true
	default:
		return false
	}
}

// Add is a helper for appending a validation error.
func (e *ValidationErrors) Add(err ValidationError) *ValidationErrors {
	if err.Field == "" || err.Reason == "" {
		return e
	}

	*e = append(*e, err)
	return e
}

// Join joins an existing Errors to this errors list.
func (e *ValidationErrors) Join(errs ValidationErrors) *ValidationErrors {
	*e = append(*e, errs...)
	return e
}

// Validate returns the result of the value's Validate method, if exists otherwise
// it adds the field and value to the error list and reports false (invalidated).
// If reason is empty, means that the field is valid, this method will return true.
func (e *ValidationErrors) Validate(field string, value interface{}) bool {
	var reason string

	if v, ok := value.(ValueValidator); ok {
		reason = v.Validate()
	}

	if reason != "" {
		e.Add(ValidationError{
			Field:  field,
			Value:  value,
			Reason: reason,
		})

		return false
	}

	return true
}

// MustBeSatisfiedFunc compares the value with the given "isEqualFunc" function and reports
// if it's valid or not. If it's not valid, a new ValidationError is added to the "e" list.
func (e *ValidationErrors) MustBeSatisfiedFunc(field string, value string, isEqualFunc func(string) bool) bool {
	if !isEqualFunc(value) {
		e.Add(ValidationError{
			Field:  field,
			Value:  value,
			Reason: "failed to satisfy constraint",
		})
		return false
	}

	return true
}

// MustBeSatisfied compares the value with the given regex and reports
// if it's valid or not. If it's not valid, a new ValidationError is added to the "e" list.
func (e *ValidationErrors) MustBeSatisfied(field string, value string, regex *regexp.Regexp) bool {
	return e.MustBeSatisfiedFunc(field, value, regex.MatchString)
}

// MustBeNotEmptyString reports and fails if the given "value" is empty.
func (e *ValidationErrors) MustBeNotEmptyString(field string, value string) bool {
	if strings.TrimSpace(value) == "" {
		e.Add(ValidationError{
			Field:  field,
			Value:  value,
			Reason: "must be not an empty string",
		})

		return false
	}

	return true
}

// MustBeInRangeString reports whether the "value" is in range of min and max.
func (e *ValidationErrors) MustBeInRangeString(field string, value string, minIncluding, maxIncluding int) bool {
	if maxIncluding <= 0 {
		maxIncluding = math.MaxInt32
	}

	if len(value) < minIncluding || len(value) > maxIncluding {
		e.Add(ValidationError{
			Field:  field,
			Value:  value,
			Reason: sprintf("characters length must be between %d and %d", minIncluding, maxIncluding),
		})
		return false
	}

	return true
}
