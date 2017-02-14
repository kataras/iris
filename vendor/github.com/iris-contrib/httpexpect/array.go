package httpexpect

import (
	"reflect"
)

// Array provides methods to inspect attached []interface{} object
// (Go representation of JSON array).
type Array struct {
	chain chain
	value []interface{}
}

// NewArray returns a new Array given a reporter used to report failures
// and value to be inspected.
//
// Both reporter and value should not be nil. If value is nil, failure is
// reported.
//
// Example:
//  array := NewArray(t, []interface{}{"foo", 123})
func NewArray(reporter Reporter, value []interface{}) *Array {
	chain := makeChain(reporter)
	if value == nil {
		chain.fail("expected non-nil array value")
	} else {
		value, _ = canonArray(&chain, value)
	}
	return &Array{chain, value}
}

// Raw returns underlying value attached to Array.
// This is the value originally passed to NewArray, converted to canonical form.
//
// Example:
//  array := NewArray(t, []interface{}{"foo", 123})
//  assert.Equal(t, []interface{}{"foo", 123.0}, array.Raw())
func (a *Array) Raw() []interface{} {
	return a.value
}

// Path is similar to Value.Path.
func (a *Array) Path(path string) *Value {
	return getPath(&a.chain, a.value, path)
}

// Schema is similar to Value.Schema.
func (a *Array) Schema(schema interface{}) *Array {
	checkSchema(&a.chain, a.value, schema)
	return a
}

// Length returns a new Number object that may be used to inspect array length.
//
// Example:
//  array := NewArray(t, []interface{}{1, 2, 3})
//  array.Length().Equal(3)
func (a *Array) Length() *Number {
	return &Number{a.chain, float64(len(a.value))}
}

// Element returns a new Value object that may be used to inspect array element
// for given index.
//
// If index is out of array bounds, Element reports failure and returns empty
// (but non-nil) value.
//
// Example:
//  array := NewArray(t, []interface{}{"foo", 123})
//  array.Element(0).String().Equal("foo")
//  array.Element(1).Number().Equal(123)
func (a *Array) Element(index int) *Value {
	if index < 0 || index >= len(a.value) {
		a.chain.fail(
			"\narray index out of bounds:\n  index %d\n\n  bounds [%d; %d)",
			index,
			0,
			len(a.value))
		return &Value{a.chain, nil}
	}
	return &Value{a.chain, a.value[index]}
}

// First returns a new Value object that may be used to inspect first element
// of given array.
//
// If given array is empty, First reports failure and returns empty
// (but non-nil) value.
//
// Example:
//  array := NewArray(t, []interface{}{"foo", 123})
//  array.First().String().Equal("foo")
func (a *Array) First() *Value {
	if len(a.value) < 1 {
		a.chain.fail("\narray is empty")
		return &Value{a.chain, nil}
	}
	return &Value{a.chain, a.value[0]}
}

// Last returns a new Value object that may be used to inspect last element
// of given array.
//
// If given array is empty, Last reports failure and returns empty
// (but non-nil) value.
//
// Example:
//  array := NewArray(t, []interface{}{"foo", 123})
//  array.Last().Number().Equal(123)
func (a *Array) Last() *Value {
	if len(a.value) < 1 {
		a.chain.fail("\narray is empty")
		return &Value{a.chain, nil}
	}
	return &Value{a.chain, a.value[len(a.value)-1]}
}

// Iter returns a new slice of Values attached to array elements.
//
// Example:
//  strings := []interface{}{"foo", "bar"}
//  array := NewArray(t, strings)
//
//  for n, val := range array.Iter() {
//      val.String().Equal(strings[n])
//  }
func (a *Array) Iter() []Value {
	if a.chain.failed() {
		return []Value{}
	}
	ret := []Value{}
	for n := range a.value {
		ret = append(ret, Value{a.chain, a.value[n]})
	}
	return ret
}

// Empty succeeds if array is empty.
//
// Example:
//  array := NewArray(t, []interface{}{})
//  array.Empty()
func (a *Array) Empty() *Array {
	return a.Equal([]interface{}{})
}

// NotEmpty succeeds if array is non-empty.
//
// Example:
//  array := NewArray(t, []interface{}{"foo", 123})
//  array.NotEmpty()
func (a *Array) NotEmpty() *Array {
	return a.NotEqual([]interface{}{})
}

// Equal succeeds if array is equal to another array.
// Before comparison, both arrays are converted to canonical form.
//
// value should be slice of any type.
//
// Example:
//  array := NewArray(t, []interface{}{"foo", 123})
//  array.Equal([]interface{}{"foo", 123})
//
//  array := NewArray(t, []interface{}{"foo", "bar"})
//  array.Equal([]string{}{"foo", "bar"})
//
//  array := NewArray(t, []interface{}{123, 456})
//  array.Equal([]int{}{123, 456})
func (a *Array) Equal(value interface{}) *Array {
	expected, ok := canonArray(&a.chain, value)
	if !ok {
		return a
	}
	if !reflect.DeepEqual(expected, a.value) {
		a.chain.fail("\nexpected array equal to:\n%s\n\nbut got:\n%s\n\ndiff:\n%s",
			dumpValue(expected),
			dumpValue(a.value),
			diffValues(expected, a.value))
	}
	return a
}

// NotEqual succeeds if array is not equal to another array.
// Before comparison, both arrays are converted to canonical form.
//
// value should be slice of any type.
//
// Example:
//  array := NewArray(t, []interface{}{"foo", 123})
//  array.NotEqual([]interface{}{123, "foo"})
func (a *Array) NotEqual(value interface{}) *Array {
	expected, ok := canonArray(&a.chain, value)
	if !ok {
		return a
	}
	if reflect.DeepEqual(expected, a.value) {
		a.chain.fail("\nexpected array not equal to:\n%s",
			dumpValue(expected))
	}
	return a
}

// Elements succeeds if array contains all given elements, in given order, and only them.
// Before comparison, array and all elements are converted to canonical form.
//
// For partial or unordered comparison, see Contains and ContainsOnly.
//
// Example:
//  array := NewArray(t, []interface{}{"foo", 123})
//  array.Elements("foo", 123)
//
// This calls are equivalent:
//  array.Elelems("a", "b")
//  array.Equal([]interface{}{"a", "b"})
func (a *Array) Elements(values ...interface{}) *Array {
	return a.Equal(values)
}

// Contains succeeds if array contains all given elements (in any order).
// Before comparison, array and all elements are converted to canonical form.
//
// Example:
//  array := NewArray(t, []interface{}{"foo", 123})
//  array.Contains(123, "foo")
func (a *Array) Contains(values ...interface{}) *Array {
	elements, ok := canonArray(&a.chain, values)
	if !ok {
		return a
	}
	for _, e := range elements {
		if !a.containsElement(e) {
			a.chain.fail("\nexpected array containing element:\n%s\n\nbut got:\n%s",
				dumpValue(e), dumpValue(a.value))
		}
	}
	return a
}

// NotContains succeeds if array contains none of given elements.
// Before comparison, array and all elements are converted to canonical form.
//
// Example:
//  array := NewArray(t, []interface{}{"foo", 123})
//  array.NotContains("bar")         // success
//  array.NotContains("bar", "foo")  // failure (array contains "foo")
func (a *Array) NotContains(values ...interface{}) *Array {
	elements, ok := canonArray(&a.chain, values)
	if !ok {
		return a
	}
	for _, e := range elements {
		if a.containsElement(e) {
			a.chain.fail("\nexpected array not containing element:\n%s\n\nbut got:\n%s",
				dumpValue(e), dumpValue(a.value))
		}
	}
	return a
}

// ContainsOnly succeeds if array contains all given elements, in any order, and only
// them. Before comparison, array and all elements are converted to canonical form.
//
// Example:
//  array := NewArray(t, []interface{}{"foo", 123})
//  array.ContainsOnly(123, "foo")
//
// This calls are equivalent:
//  array.ContainsOnly("a", "b")
//  array.ContainsOnly("b", "a")
func (a *Array) ContainsOnly(values ...interface{}) *Array {
	elements, ok := canonArray(&a.chain, values)
	if !ok {
		return a
	}
	if len(elements) != len(a.value) {
		a.chain.fail("\nexpected array of length == %d:\n%s\n\n"+
			"but got array of length %d:\n%s",
			len(elements), dumpValue(elements),
			len(a.value), dumpValue(a.value))
		return a
	}
	for _, e := range elements {
		if !a.containsElement(e) {
			a.chain.fail("\nexpected array containing element:\n%s\n\nbut got:\n%s",
				dumpValue(e), dumpValue(a.value))
		}
	}
	return a
}

func (a *Array) containsElement(expected interface{}) bool {
	for _, e := range a.value {
		if reflect.DeepEqual(expected, e) {
			return true
		}
	}
	return false
}
