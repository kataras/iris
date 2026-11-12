package validation

import (
	"fmt"

	"github.com/kataras/iris/v14/core/pkg/constraints"
)

// NumberType is a type constraint that accepts any numeric type.
type NumberType interface {
	constraints.Number
}

// NumberError describes a number field validation error.
type NumberError[T NumberType] struct{ *FieldError[T] }

// Number returns a new number validation error.
func Number[T NumberType](field string, value T) *NumberError[T] {
	return &NumberError[T]{Field(field, value)}
}

// Positive adds an error if the value is not positive.
func (e *NumberError[T]) Positive() *NumberError[T] {
	e.Func(Positive)
	return e
}

// Negative adds an error if the value is not negative.
func (e *NumberError[T]) Negative() *NumberError[T] {
	e.Func(Negative)
	return e
}

// Zero adds an error if the value is not zero.
func (e *NumberError[T]) Zero() *NumberError[T] {
	e.Func(Zero)
	return e
}

// NonZero adds an error if the value is zero.
func (e *NumberError[T]) NonZero() *NumberError[T] {
	e.Func(NonZero)
	return e
}

// InRange adds an error if the value is not in the range.
func (e *NumberError[T]) InRange(min, max T) *NumberError[T] {
	e.Func(InRange(min, max))
	return e
}

// Positive accepts any numeric type and
// returns a message if the value is not positive.
func Positive[T NumberType](n T) string {
	if n <= 0 {
		return "must be positive"
	}

	return ""
}

// Negative accepts any numeric type and returns a message if the value is not negative.
func Negative[T NumberType](n T) string {
	if n >= 0 {
		return "must be negative"
	}

	return ""
}

// Zero accepts any numeric type and returns a message if the value is not zero.
func Zero[T NumberType](n T) string {
	if n != 0 {
		return "must be zero"
	}

	return ""
}

// NonZero accepts any numeric type and returns a message if the value is zero.
func NonZero[T NumberType](n T) string {
	if n == 0 {
		return "must not be zero"
	}

	return ""
}

// InRange accepts any numeric type and returns a message if the value is not in the range.
func InRange[T NumberType](min, max T) func(T) string {
	return func(n T) string {
		// Special case: if both min and max are 0, we interpret as "no constraints".
		if min == 0 && max == 0 {
			return ""
		}

		if min == max { // equal.
			if n != min {
				return fmt.Sprintf("must be %v", min)
			}

			return ""
		}

		if max == 0 && min > 0 { // only min provided (no maximum).
			if n < min {
				return fmt.Sprintf("must be at least %v", min)
			}
			return ""
		}

		if min == 0 && max > 0 { // only max provided (no minimum).
			if n > max {
				return fmt.Sprintf("must be at most %v", max)
			}
			return ""
		}

		// Both min and max provided
		if n < min || n > max {
			return fmt.Sprintf("must be between %v and %v", min, max)
		}

		return ""
	}
}

// FormatRange returns a string representation of a range of values, such as "[1, 10]".
// It uses the NumberType constraint, which means that the parameters must be numeric types
// that support comparison and formatting operations.
func FormatRange[T NumberType](min, max T) string {
	return fmt.Sprintf("[%v, %v]", min, max)
}
