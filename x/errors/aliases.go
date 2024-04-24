package errors

import (
	"errors"
	"fmt"
)

var (
	// Is is an alias of the standard errors.Is function.
	Is = errors.Is
	// As is an alias of the standard errors.As function.
	As = errors.As
	// New is an alias of the standard errors.New function.
	New = errors.New
	// Unwrap is an alias of the standard errors.Unwrap function.
	Unwrap = errors.Unwrap
	// Join is an alias of the standard errors.Join function.
	Join = errors.Join
)

func sprintf(format string, args ...interface{}) string {
	if len(args) > 0 {
		return fmt.Sprintf(format, args...)
	}

	return format
}
