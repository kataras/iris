package errors

import (
	"strconv"
	"strings"
)

// ValidationError defines the interface for field validation errors that can be
// automatically processed by the error handling system. Custom error types that
// implement this interface join the validation error processing pipeline with no
// extra registration.
//
// ValidationError provides a structured way to represent field-level validation
// failures with detailed information about what went wrong. The interface is
// designed to work with ErrorCodeName's Validation and Err methods for automatic
// error response generation.
//
// Example implementation:
//
//	type CustomValidationError struct {
//	    Field   string
//	    Value   any
//	    Message string
//	}
//
//	func (e CustomValidationError) Error() string {
//	    return fmt.Sprintf("validation failed for field %s: %s", e.Field, e.Message)
//	}
//
//	func (e CustomValidationError) GetReason() string {
//	    return e.Message
//	}
//
//	// Usage with error handling
//	validationErr := CustomValidationError{
//	    Field: "email",
//	    Value: "invalid-email",
//	    Message: "must be a valid email address",
//	}
//
//	errors.InvalidArgument.Validation(ctx, validationErr)
//	return
//
// The interface intentionally focuses on the validation reason rather than
// field name and value to maintain flexibility while ensuring that the
// essential validation information is available for error processing.
type ValidationError interface {
	error

	// GetReason returns a human-readable description of why the validation failed.
	// This message should be clear and actionable, helping users understand
	// what needs to be corrected. The reason is used in error responses and
	// logging to provide meaningful feedback about validation failures.
	//
	// Example reasons:
	//  - "must be a valid email address"
	//  - "length must be between 8 and 32 characters"
	//  - "must contain at least one uppercase letter"
	//  - "value must be greater than 0"
	GetReason() string
}

// ValidationErrors represents a collection of validation errors that can occur
// during the validation of multiple fields or complex data structures. It
// implements the error interface and provides a formatted representation of
// all validation failures.
//
// ValidationErrors is particularly useful when validating forms, API payloads,
// or configuration data where multiple fields might have validation errors
// simultaneously. The error message format includes indexing to help identify
// specific validation failures.
//
// Example usage:
//
//	var validationErrors ValidationErrors
//
//	if emailErr := validateEmail(user.Email); emailErr != nil {
//	    validationErrors = append(validationErrors, emailErr)
//	}
//
//	if passwordErr := validatePassword(user.Password); passwordErr != nil {
//	    validationErrors = append(validationErrors, passwordErr)
//	}
//
//	if len(validationErrors) > 0 {
//	    errors.InvalidArgument.Validation(ctx, validationErrors...)
//	    return
//	}
//
// The resulting error message format is: "[0] first error, [1] second error"
// where each validation error is indexed and separated by commas.
type ValidationErrors []ValidationError

// Error implements the error interface for ValidationErrors, providing a
// formatted string representation of all validation errors in the collection.
// Each error is prefixed with its index in square brackets and separated
// by commas for easy parsing and readability.
//
// The format follows the pattern: "[index] error message, [index] error message"
// This indexing helps developers and users identify specific validation
// failures when multiple errors occur simultaneously.
//
// Example output:
//
//	"[0] email must be a valid email address, [1] password must be at least 8 characters"
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
