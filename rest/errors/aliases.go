package errors

import (
	"errors"
	"fmt"
)

var (
	// Is reports whether any error in err's tree matches target. It is an alias
	// of the standard errors.Is function, provided for convenience when working
	// within the errors package to avoid import conflicts.
	//
	// The tree consists of err itself followed by the sequence of errors obtained by
	// repeatedly calling Unwrap. When err wraps multiple errors, Is examines err
	// followed by a depth-first traversal of its children.
	//
	// An error is considered to match a target if it is equal to that target or if
	// it implements a method Is(error) bool such that Is(target) returns true.
	//
	// Example usage:
	//
	//	if errors.Is(err, io.EOF) {
	//	    // Handle end of file
	//	}
	Is = errors.Is

	// As finds the first error in err's tree that matches target, and if one is found,
	// sets target to that error value and returns true. It is an alias of the standard
	// errors.As function, provided for convenience within the errors package.
	//
	// The tree consists of err itself followed by the sequence of errors obtained by
	// repeatedly calling Unwrap. When err wraps multiple errors, As examines err
	// followed by a depth-first traversal of its children.
	//
	// An error matches target if the error's concrete value is assignable to the value
	// pointed to by target, or if the error has a method As(any) bool such that
	// As(target) returns true.
	//
	// Example usage:
	//
	//	var pathError *fs.PathError
	//	if errors.As(err, &pathError) {
	//	    fmt.Println("Failed at path:", pathError.Path)
	//	}
	As = errors.As

	// New returns an error that formats as the given text. It is an alias of the
	// standard errors.New function, provided for convenience within the errors package.
	//
	// Each call to New returns a distinct error value even if the text is identical.
	//
	// Example usage:
	//
	//	err := errors.New("something went wrong")
	New = errors.New

	// Unwrap returns the result of calling the Unwrap method on err, if err's type contains
	// an Unwrap method returning error. It is an alias of the standard errors.Unwrap function,
	// provided for convenience within the errors package.
	//
	// Otherwise, Unwrap returns nil.
	//
	// Example usage:
	//
	//	if wrappedErr := errors.Unwrap(err); wrappedErr != nil {
	//	    // Handle the underlying error
	//	}
	Unwrap = errors.Unwrap

	// Join returns an error that wraps the given errors. It is an alias of the standard
	// errors.Join function, provided for convenience within the errors package.
	//
	// Any nil error values are discarded. Join returns nil if every value in errs is nil.
	// The error formats as the concatenation of the strings obtained by calling the Error
	// method of each element of errs, with a newline between each string.
	//
	// A non-nil error returned by Join implements the Unwrap() []error method.
	//
	// Example usage:
	//
	//	err1 := errors.New("first error")
	//	err2 := errors.New("second error")
	//	combined := errors.Join(err1, err2)
	Join = errors.Join
)

// AsType is the standard library's errors.AsType.
//
// It finds the first error in err's tree that is an E and returns it, with no
// target variable to declare first:
//
//	if quota, ok := errors.AsType[*QuotaError](err); ok {
//		fmt.Println("the limit is", quota.Limit)
//	}
//
// It is a function and not a variable like Is and As above, because a generic
// function cannot be assigned without naming its type argument.
func AsType[E error](err error) (E, bool) {
	return errors.AsType[E](err)
}

// UnwrapAll recursively unwraps an error and returns a slice containing all errors
// in the wrapping chain, starting with the top-level error. This function handles
// both single-error wrapping (via Unwrap() error) and multi-error wrapping
// (via Unwrap() []error), so the whole error tree is visible.
//
// The function traverses the error chain depth-first, collecting all errors
// encountered. For errors that wrap multiple errors (implementing Unwrap() []error),
// it recursively processes each wrapped error.
//
// Example usage:
//
//	// Single error chain
//	err1 := errors.New("root cause")
//	err2 := fmt.Errorf("wrapper: %w", err1)
//	err3 := fmt.Errorf("top level: %w", err2)
//
//	allErrors := errors.UnwrapAll(err3)
//	// allErrors contains: [err3, err2, err1]
//
//	// Multi-error case
//	err1 := errors.New("first error")
//	err2 := errors.New("second error")
//	combined := errors.Join(err1, err2)
//
//	allErrors := errors.UnwrapAll(combined)
//	// allErrors contains: [combined, err1, err2]
//
// Returns an empty slice if the input error is nil.
func UnwrapAll(err error) (list []error) {
	for err != nil {
		list = append(list, err)

		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			// list = append(list, UnwrapAll(err)...)
		case interface{ Unwrap() []error }:
			for _, nestedErr := range x.Unwrap() {
				list = append(list, UnwrapAll(nestedErr)...)
			}
			return list
		default:
			err = nil
		}
	}

	return list
}

// sprintf is a utility function that safely formats strings using fmt.Sprintf.
// If no arguments are provided, it returns the format string unchanged,
// avoiding unnecessary formatting overhead. This is used internally by the
// errors package for efficient string formatting in error messages.
//
// This function is particularly useful when building error messages where
// the format string might or might not have placeholders, allowing for
// both static and dynamic error message construction.
func sprintf(format string, args ...any) string {
	if len(args) > 0 {
		return fmt.Sprintf(format, args...)
	}

	return format
}
