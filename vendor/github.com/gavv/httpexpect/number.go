package httpexpect

import (
	"math"
)

// Number provides methods to inspect attached float64 value
// (Go representation of JSON number).
type Number struct {
	chain chain
	value float64
}

// NewNumber returns a new Number given a reporter used to report
// failures and value to be inspected.
//
// reporter should not be nil.
//
// Example:
//  number := NewNumber(t, 123.4)
func NewNumber(reporter Reporter, value float64) *Number {
	return &Number{makeChain(reporter), value}
}

// Raw returns underlying value attached to Number.
// This is the value originally passed to NewNumber.
//
// Example:
//  number := NewNumber(t, 123.4)
//  assert.Equal(t, 123.4, number.Raw())
func (n *Number) Raw() float64 {
	return n.value
}

// Path is similar to Value.Path.
func (n *Number) Path(path string) *Value {
	return getPath(&n.chain, n.value, path)
}

// Schema is similar to Value.Schema.
func (n *Number) Schema(schema interface{}) *Number {
	checkSchema(&n.chain, n.value, schema)
	return n
}

// Equal succeeds if number is equal to given value.
//
// value should have numeric type convertible to float64. Before comparison,
// it is converted to float64.
//
// Example:
//  number := NewNumber(t, 123)
//  number.Equal(float64(123))
//  number.Equal(int32(123))
func (n *Number) Equal(value interface{}) *Number {
	v, ok := canonNumber(&n.chain, value)
	if !ok {
		return n
	}
	if !(n.value == v) {
		n.chain.fail("\nexpected number equal to:\n %v\n\nbut got:\n %v",
			v, n.value)
	}
	return n
}

// NotEqual succeeds if number is not equal to given value.
//
// value should have numeric type convertible to float64. Before comparison,
// it is converted to float64.
//
// Example:
//  number := NewNumber(t, 123)
//  number.NotEqual(float64(321))
//  number.NotEqual(int32(321))
func (n *Number) NotEqual(value interface{}) *Number {
	v, ok := canonNumber(&n.chain, value)
	if !ok {
		return n
	}
	if !(n.value != v) {
		n.chain.fail("\nexpected number not equal to:\n %v\n\nbut got:\n %v",
			v, n.value)
	}
	return n
}

// EqualDelta succeeds if two numerals are within delta of each other.
//
// Example:
//  number := NewNumber(t, 123.0)
//  number.EqualDelta(123.2, 0.3)
func (n *Number) EqualDelta(value, delta float64) *Number {
	if math.IsNaN(n.value) || math.IsNaN(value) || math.IsNaN(delta) {
		n.chain.fail("\nexpected number equal to:\n %v\n\nbut got:\n %v\n\ndelta:\n %v",
			value, n.value, delta)
		return n
	}

	diff := (n.value - value)

	if diff < -delta || diff > delta {
		n.chain.fail("\nexpected number equal to:\n %v\n\nbut got:\n %v\n\ndelta:\n %v",
			value, n.value, delta)
		return n
	}

	return n
}

// NotEqualDelta succeeds if two numerals are not within delta of each other.
//
// Example:
//  number := NewNumber(t, 123.0)
//  number.NotEqualDelta(123.2, 0.1)
func (n *Number) NotEqualDelta(value, delta float64) *Number {
	if math.IsNaN(n.value) || math.IsNaN(value) || math.IsNaN(delta) {
		n.chain.fail(
			"\nexpected number not equal to:\n %v\n\nbut got:\n %v\n\ndelta:\n %v",
			value, n.value, delta)
		return n
	}

	diff := (n.value - value)

	if !(diff < -delta || diff > delta) {
		n.chain.fail(
			"\nexpected number not equal to:\n %v\n\nbut got:\n %v\n\ndelta:\n %v",
			value, n.value, delta)
		return n
	}

	return n
}

// Gt succeeds if number is greater than given value.
//
// value should have numeric type convertible to float64. Before comparison,
// it is converted to float64.
//
// Example:
//  number := NewNumber(t, 123)
//  number.Gt(float64(122))
//  number.Gt(int32(122))
func (n *Number) Gt(value interface{}) *Number {
	v, ok := canonNumber(&n.chain, value)
	if !ok {
		return n
	}
	if !(n.value > v) {
		n.chain.fail("\nexpected number > then:\n %v\n\nbut got:\n %v",
			v, n.value)
	}
	return n
}

// Ge succeeds if number is greater than or equal to given value.
//
// value should have numeric type convertible to float64. Before comparison,
// it is converted to float64.
//
// Example:
//  number := NewNumber(t, 123)
//  number.Ge(float64(122))
//  number.Ge(int32(122))
func (n *Number) Ge(value interface{}) *Number {
	v, ok := canonNumber(&n.chain, value)
	if !ok {
		return n
	}
	if !(n.value >= v) {
		n.chain.fail("\nexpected number >= then:\n %v\n\nbut got:\n %v",
			v, n.value)
	}
	return n
}

// Lt succeeds if number is lesser than given value.
//
// value should have numeric type convertible to float64. Before comparison,
// it is converted to float64.
//
// Example:
//  number := NewNumber(t, 123)
//  number.Lt(float64(124))
//  number.Lt(int32(124))
func (n *Number) Lt(value interface{}) *Number {
	v, ok := canonNumber(&n.chain, value)
	if !ok {
		return n
	}
	if !(n.value < v) {
		n.chain.fail("\nexpected number < then:\n %v\n\nbut got:\n %v",
			v, n.value)
	}
	return n
}

// Le succeeds if number is lesser than or equal to given value.
//
// value should have numeric type convertible to float64. Before comparison,
// it is converted to float64.
//
// Example:
//  number := NewNumber(t, 123)
//  number.Le(float64(124))
//  number.Le(int32(124))
func (n *Number) Le(value interface{}) *Number {
	v, ok := canonNumber(&n.chain, value)
	if !ok {
		return n
	}
	if !(n.value <= v) {
		n.chain.fail("\nexpected number <= then:\n %v\n\nbut got:\n %v",
			v, n.value)
	}
	return n
}

// InRange succeeds if number is in given range [min; max].
//
// min and max should have numeric type convertible to float64. Before comparison,
// they are converted to float64.
//
// Example:
//  number := NewNumber(t, 123)
//  number.InRange(float32(100), int32(200))  // success
//  number.InRange(100, 200)                  // success
//  number.InRange(123, 123)                  // success
func (n *Number) InRange(min, max interface{}) *Number {
	a, ok := canonNumber(&n.chain, min)
	if !ok {
		return n
	}
	b, ok := canonNumber(&n.chain, max)
	if !ok {
		return n
	}
	if !(n.value >= a && n.value <= b) {
		n.chain.fail("\nexpected number in range:\n [%v; %v]\n\nbut got:\n %v",
			a, b, n.value)
	}
	return n
}
