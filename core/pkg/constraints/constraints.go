// Package constraints defines the type sets the framework's generic code is
// bounded by. It mirrors the subset of golang.org/x/exp/constraints that Iris
// used, so the module does not depend on the experimental repository for
// six one-line interfaces.
package constraints

// Signed is the set of signed integer types.
type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// Unsigned is the set of unsigned integer types.
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Integer is the set of integer types.
type Integer interface {
	Signed | Unsigned
}

// Float is the set of floating-point types.
type Float interface {
	~float32 | ~float64
}

// Number is the set of integer and floating-point types.
type Number interface {
	Integer | Float
}

// Ordered is the set of types that support the < <= >= > operators.
type Ordered interface {
	Integer | Float | ~string
}
