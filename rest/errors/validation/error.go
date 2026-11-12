// Package validation provides generic field validation utilities for the Iris web framework.
// It offers type-safe validation functions and error handling for common validation scenarios
// including string, numeric, slice, and custom validation patterns.
package validation

import (
	"fmt"

	"github.com/kataras/iris/v14/rest/errors"
)

// FieldError represents a generic field validation error with type safety for the field value.
// It implements the errors.ValidationError interface and reports the field name, the
// actual value and the reason the value was rejected.
//
// FieldError uses Go generics to maintain type safety for field values while providing
// a consistent validation error structure. This allows for precise error reporting and
// debugging while maintaining compile-time type checking.
//
// Example usage:
//
//	// String field validation
//	emailField := validation.Field("email", "invalid-email")
//	emailField.Func(validation.Email)
//	if err := emailField.Get(); err != nil {
//	    return errors.BadRequest.Validation(err)
//	}
//
//	// Numeric field validation
//	ageField := validation.Field("age", -5)
//	ageField.Func(validation.InRange(0, 150))
//	if err := ageField.Get(); err != nil {
//	    return errors.BadRequest.Validation(err)
//	}
//
//	// Custom validation
//	passwordField := validation.Field("password", "123")
//	passwordField.Func(func(value string) string {
//	    if len(value) < 8 {
//	        return "must be at least 8 characters long"
//	    }
//	    return ""
//	})
//
// The FieldError struct can be serialized to JSON for API responses, providing
// structured error information to clients.
type FieldError[T any] struct {
	// Field contains the name of the field being validated.
	// This helps identify which specific field failed validation in complex data structures.
	Field string `json:"field"`

	// Value contains the actual value that failed validation.
	// The type T ensures type safety and allows for precise error reporting.
	Value T `json:"value"`

	// Reason contains the human-readable explanation of why validation failed.
	// This message should be clear and actionable for end users.
	Reason string `json:"reason"`
}

// Field creates a new FieldError instance for the specified field name and value.
// This is the primary constructor for field validation and should be used as the
// starting point for all field validation operations.
//
// The function uses Go generics to ensure type safety for the field value,
// allowing validation functions to work with strongly-typed values while
// maintaining error reporting consistency.
//
// Example usage:
//
//	// Create field validators for different types
//	emailField := validation.Field("email", user.Email)
//	ageField := validation.Field("age", user.Age)
//	tagsField := validation.Field("tags", user.Tags)
//
//	// Chain validation functions
//	emailField.Func(validation.NotEmpty, validation.Email)
//	ageField.Func(validation.InRange(18, 99))
//	tagsField.Func(validation.SliceLength[string, []string](1, 10))
//
// After creating a field validator, use the Func method to apply validation
// rules, then call Get() to retrieve any validation errors.
func Field[T any](field string, value T) *FieldError[T] {
	return &FieldError[T]{Field: field, Value: value}
}

// Error implements the standard error interface, providing a formatted string
// representation of the validation failure. The error message includes the
// field name, the actual value and the reason, which is what you need when debugging.
//
// The format follows the pattern:
//
//	field "fieldname" got invalid value of actualvalue: reason: validation reason
//
// Example output:
//
//	field "email" got invalid value of invalid-email: reason: must be a valid email address
//	field "age" got invalid value of -5: reason: must be greater than or equal to 0
func (e *FieldError[T]) Error() string {
	return fmt.Sprintf("field %q got invalid value of %v: reason: %s", e.Field, e.Value, e.Reason)
}

// Get returns the FieldError as an error if validation failed, or nil if validation passed.
// This method should be called after applying validation functions to check if any
// validation rules were violated.
//
// The method returns nil if the FieldError has no validation reason, indicating
// that all validation rules passed successfully. This allows for clean error
// handling patterns where only failed validations propagate as errors.
//
// Example usage:
//
//	emailField := validation.Field("email", user.Email)
//	emailField.Func(validation.NotEmpty, validation.Email)
//
//	if err := emailField.Get(); err != nil {
//	    return errors.BadRequest.Validation(err)
//	}
//
//	// Validation passed, continue processing
//	return nil
func (e *FieldError[T]) Get() error {
	if e.IsZero() {
		return nil
	}

	return e
}

// GetField returns the name of the field being validated.
// This method is part of the extended ValidationError interface and provides
// access to field metadata for error processing and response generation.
//
// Field names should follow consistent naming conventions to ensure
// predictable error handling and client-side error processing.
func (e *FieldError[T]) GetField() string {
	return e.Field
}

// GetValue returns the actual value that failed validation as an any.
// This method provides access to the field value for logging, debugging,
// and custom error processing scenarios.
//
// The value is returned as any to satisfy the ValidationError interface
// while maintaining the original type information within the FieldError struct.
func (e *FieldError[T]) GetValue() any {
	return e.Value
}

// GetReason implements the errors.ValidationError interface, returning the
// human-readable explanation of why validation failed. This message should
// be suitable for display to end users and provide clear guidance on how
// to correct the validation error.
//
// Example reasons:
//   - "is required"
//   - "must be a valid email address"
//   - "must be between 8 and 32 characters"
//   - "must contain at least one uppercase letter"
func (e *FieldError[T]) GetReason() string {
	return e.Reason
}

// IsZero reports whether the FieldError represents a successful validation.
// A FieldError is considered "zero" (successful) if it is nil or has an empty
// validation reason, indicating that no validation rules were violated.
//
// This method is used internally by Get() to determine whether to return
// an error and can be used in validation logic to check validation status
// without converting to an error interface.
//
// Example usage:
//
//	field := validation.Field("username", user.Username)
//	field.Func(validation.NotEmpty, validation.StringLength(3, 0))
//
//	if !field.IsZero() {
//	    log.Printf("Validation failed: %s", field.GetReason())
//	}
func (e *FieldError[T]) IsZero() bool {
	return e == nil || e.Reason == ""
}

// joinReason appends a validation reason to the existing reason string.
// Multiple validation failures are joined with commas, so one field can report every
// problem it has at once.
//
// This method is used internally by the Func method when multiple validation
// functions are applied to the same field. It ensures that all validation
// failures are captured and reported in a single, readable message.
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

// Func applies one or more validation functions to the field value and accumulates
// any validation failure reasons. Each validation function receives the field value
// and returns an empty string for successful validation or an error message for failures.
//
// Multiple validation functions can be chained together, and all failure reasons
// are joined with commas, so a field that breaks several rules reports all of
// them at once.
//
// The method returns the FieldError instance to enable method chaining for
// fluent validation syntax.
//
// Example usage:
//
//	// Single validation function
//	emailField := validation.Field("email", user.Email)
//	emailField.Func(validation.Email)
//
//	// Multiple validation functions
//	passwordField := validation.Field("password", user.Password)
//	passwordField.Func(
//	    validation.NotEmpty,
//	    validation.StringLength(8, 128),
//	    func(value string) string {
//	        return validation.StrongPassword(value, validation.PasswordStrong)
//	    },
//	)
//
//	// Custom validation functions
//	usernameField := validation.Field("username", user.Username)
//	usernameField.Func(
//	    validation.NotEmpty,
//	    func(value string) string {
//	        if strings.Contains(value, " ") {
//	            return "must not contain spaces"
//	        }
//	        return ""
//	    },
//	    func(value string) string {
//	        if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(value) {
//	            return "must contain only letters, numbers, and underscores"
//	        }
//	        return ""
//	    },
//	)
//
// If multiple validation functions fail, their reasons are joined:
//
//	"is required, must be at least 8 characters, must contain at least one uppercase letter"
func (e *FieldError[T]) Func(fns ...func(value T) string) *FieldError[T] {
	for _, fn := range fns {
		e.joinReason(fn(e.Value))
	}

	return e
}

// Join combines multiple validation errors into a single error, filtering out
// nil errors and errors with empty reasons. This function is essential for
// collecting validation results from multiple fields and presenting them as
// a unified validation failure.
//
// The function returns the standard error interface rather than ValidationError
// to ensure proper nil handling. When all validation errors are nil or empty,
// Join returns nil instead of an empty ValidationErrors slice, preventing
// false positive error conditions.
//
// Example usage:
//
//	// Validate multiple fields
//	emailErr := validation.Field("email", user.Email).Func(validation.Email)
//	passwordErr := validation.Field("password", user.Password).Func(validation.NotEmpty)
//	ageErr := validation.Field("age", user.Age).Func(validation.InRange(0, 150))
//
//	// Combine all validation errors
//	if err := validation.Join(emailErr, passwordErr, ageErr); err != nil {
//	    return errors.BadRequest.Validation(err)
//	}
//
//	// All validations passed
//	return nil
//
// When multiple errors are present, the resulting error message will be formatted as:
//
//	"[0] email validation error, [1] password validation error, [2] age validation error"
//
// This function is particularly useful in complex validation scenarios where
// multiple fields need to be validated independently, and all validation
// failures should be reported together for better user experience.
func Join(errs ...errors.ValidationError) error {
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
