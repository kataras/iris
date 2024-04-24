package reflex

import "reflect"

// IsError reports whether "typ" is an error type.
func IsError(typ interface{ Implements(reflect.Type) bool }) bool {
	return typ.Implements(ErrTyp)
}
