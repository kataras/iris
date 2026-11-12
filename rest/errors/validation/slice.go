package validation

import "fmt"

// SliceType is a type constraint that accepts any slice type.
type SliceType[T any] interface {
	~[]T
}

// SliceError describes a slice field validation error.
type SliceError[T any, V SliceType[T]] struct{ *FieldError[V] }

// Slice returns a new slice validation error.
func Slice[T any, V SliceType[T]](field string, value V) *SliceError[T, V] {
	return &SliceError[T, V]{Field(field, value)}
}

// NotEmpty adds an error if the slice is empty.
func (e *SliceError[T, V]) NotEmpty() *SliceError[T, V] {
	e.Func(NotEmptySlice)
	return e
}

// Length adds an error if the slice length is not in the given range.
func (e *SliceError[T, V]) Length(min, max int) *SliceError[T, V] {
	e.Func(SliceLength[T, V](min, max))
	return e
}

// NotEmptySlice accepts any slice and returns a message if the value is empty.
func NotEmptySlice[T any, V SliceType[T]](s V) string {
	if len(s) == 0 {
		return "must not be empty"
	}

	return ""
}

// SliceLength accepts any slice and returns a message if the length is not in the given range.
func SliceLength[T any, V SliceType[T]](min, max int) func(s V) string {
	return func(s V) string {
		n := len(s)

		if min == max {
			if n != min {
				return fmt.Sprintf("must be %d elements", min)
			}

			return ""
		}

		if n < min || n > max {
			return fmt.Sprintf("must be between %d and %d elements", min, max)
		}

		return ""
	}
}
