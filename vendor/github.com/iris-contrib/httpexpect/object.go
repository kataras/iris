package httpexpect

import (
	"reflect"
)

// Object provides methods to inspect attached map[string]interface{} object
// (Go representation of JSON object).
type Object struct {
	chain chain
	value map[string]interface{}
}

// NewObject returns a new Object given a reporter used to report failures
// and value to be inspected.
//
// Both reporter and value should not be nil. If value is nil, failure is
// reported.
//
// Example:
//  object := NewObject(t, map[string]interface{}{"foo": 123})
func NewObject(reporter Reporter, value map[string]interface{}) *Object {
	chain := makeChain(reporter)
	if value == nil {
		chain.fail("expected non-nil map value")
	} else {
		value, _ = canonMap(&chain, value)
	}
	return &Object{chain, value}
}

// Raw returns underlying value attached to Object.
// This is the value originally passed to NewObject, converted to canonical form.
//
// Example:
//  object := NewObject(t, map[string]interface{}{"foo": 123})
//  assert.Equal(t, map[string]interface{}{"foo": 123.0}, object.Raw())
func (o *Object) Raw() map[string]interface{} {
	return o.value
}

// Path is similar to Value.Path.
func (o *Object) Path(path string) *Value {
	return getPath(&o.chain, o.value, path)
}

// Schema is similar to Value.Schema.
func (o *Object) Schema(schema interface{}) *Object {
	checkSchema(&o.chain, o.value, schema)
	return o
}

// Keys returns a new Array object that may be used to inspect objects keys.
//
// Example:
//  object := NewObject(t, map[string]interface{}{"foo": 123, "bar": 456})
//  object.Keys().ContainsOnly("foo", "bar")
func (o *Object) Keys() *Array {
	keys := []interface{}{}
	for k := range o.value {
		keys = append(keys, k)
	}
	return &Array{o.chain, keys}
}

// Values returns a new Array object that may be used to inspect objects values.
//
// Example:
//  object := NewObject(t, map[string]interface{}{"foo": 123, "bar": 456})
//  object.Values().ContainsOnly(123, 456)
func (o *Object) Values() *Array {
	values := []interface{}{}
	for _, v := range o.value {
		values = append(values, v)
	}
	return &Array{o.chain, values}
}

// Value returns a new Value object that may be used to inspect single value
// for given key.
//
// Example:
//  object := NewObject(t, map[string]interface{}{"foo": 123})
//  object.Value("foo").Number().Equal(123)
func (o *Object) Value(key string) *Value {
	value, ok := o.value[key]
	if !ok {
		o.chain.fail("\nexpected object containing key '%s', but got:\n%s",
			key, dumpValue(o.value))
		return &Value{o.chain, nil}
	}
	return &Value{o.chain, value}
}

// Empty succeeds if object is empty.
//
// Example:
//  object := NewObject(t, map[string]interface{}{})
//  object.Empty()
func (o *Object) Empty() *Object {
	return o.Equal(map[string]interface{}{})
}

// NotEmpty succeeds if object is non-empty.
//
// Example:
//  object := NewObject(t, map[string]interface{}{"foo": 123})
//  object.NotEmpty()
func (o *Object) NotEmpty() *Object {
	return o.NotEqual(map[string]interface{}{})
}

// Equal succeeds if object is equal to another object.
// Before comparison, both objects are converted to canonical form.
//
// value should map[string]interface{} or struct.
//
// Example:
//  object := NewObject(t, map[string]interface{}{"foo": 123})
//  object.Equal(map[string]interface{}{"foo": 123})
func (o *Object) Equal(value interface{}) *Object {
	expected, ok := canonMap(&o.chain, value)
	if !ok {
		return o
	}
	if !reflect.DeepEqual(expected, o.value) {
		o.chain.fail("\nexpected object equal to:\n%s\n\nbut got:\n%s\n\ndiff:\n%s",
			dumpValue(expected),
			dumpValue(o.value),
			diffValues(expected, o.value))
	}
	return o
}

// NotEqual succeeds if object is not equal to another object.
// Before comparison, both objects are converted to canonical form.
//
// value should map[string]interface{} or struct.
//
// Example:
//  object := NewObject(t, map[string]interface{}{"foo": 123})
//  object.Equal(map[string]interface{}{"bar": 123})
func (o *Object) NotEqual(v interface{}) *Object {
	expected, ok := canonMap(&o.chain, v)
	if !ok {
		return o
	}
	if reflect.DeepEqual(expected, o.value) {
		o.chain.fail("\nexpected object not equal to:\n%s",
			dumpValue(expected))
	}
	return o
}

// ContainsKey succeeds if object contains given key.
//
// Example:
//  object := NewObject(t, map[string]interface{}{"foo": 123})
//  object.ContainsKey("foo")
func (o *Object) ContainsKey(key string) *Object {
	if !o.containsKey(key) {
		o.chain.fail("\nexpected object containing key '%s', but got:\n%s",
			key, dumpValue(o.value))
	}
	return o
}

// NotContainsKey succeeds if object doesn't contain given key.
//
// Example:
//  object := NewObject(t, map[string]interface{}{"foo": 123})
//  object.NotContainsKey("bar")
func (o *Object) NotContainsKey(key string) *Object {
	if o.containsKey(key) {
		o.chain.fail(
			"\nexpected object not containing key '%s', but got:\n%s", key,
			dumpValue(o.value))
	}
	return o
}

// ContainsMap succeeds if object contains given sub-object.
// Before comparison, both objects are converted to canonical form.
//
// value should map[string]interface{} or struct.
//
// Example:
//  object := NewObject(t, map[string]interface{}{
//      "foo": 123,
//      "bar": []interface{}{"x", "y"},
//      "bar": map[string]interface{}{
//          "a": true,
//          "b": false,
//      },
//  })
//
//  object.ContainsMap(map[string]interface{}{  // success
//      "foo": 123,
//      "bar": map[string]interface{}{
//          "a": true,
//      },
//  })
//
//  object.ContainsMap(map[string]interface{}{  // failure
//      "foo": 123,
//      "qux": 456,
//  })
//
//  object.ContainsMap(map[string]interface{}{  // failure, slices should match exactly
//      "bar": []interface{}{"x"},
//  })
func (o *Object) ContainsMap(value interface{}) *Object {
	if !o.containsMap(value) {
		o.chain.fail("\nexpected object containing sub-object:\n%s\n\nbut got:\n%s",
			dumpValue(value), dumpValue(o.value))
	}
	return o
}

// NotContainsMap succeeds if object doesn't contain given sub-object exactly.
// Before comparison, both objects are converted to canonical form.
//
// value should map[string]interface{} or struct.
//
// Example:
//  object := NewObject(t, map[string]interface{}{"foo": 123, "bar": 456})
//  object.NotContainsMap(map[string]interface{}{"foo": 123, "bar": "no-no-no"})
func (o *Object) NotContainsMap(value interface{}) *Object {
	if o.containsMap(value) {
		o.chain.fail("\nexpected object not containing sub-object:\n%s\n\nbut got:\n%s",
			dumpValue(value), dumpValue(o.value))
	}
	return o
}

// ValueEqual succeeds if object's value for given key is equal to given value.
// Before comparison, both values are converted to canonical form.
//
// value should map[string]interface{} or struct.
//
// Example:
//  object := NewObject(t, map[string]interface{}{"foo": 123})
//  object.ValueEqual("foo", 123)
func (o *Object) ValueEqual(key string, value interface{}) *Object {
	if !o.containsKey(key) {
		o.chain.fail("\nexpected object containing key '%s', but got:\n%s",
			key, dumpValue(o.value))
		return o
	}
	expected, ok := canonValue(&o.chain, value)
	if !ok {
		return o
	}
	if !reflect.DeepEqual(expected, o.value[key]) {
		o.chain.fail(
			"\nexpected value for key '%s' equal to:\n%s\n\nbut got:\n%s\n\ndiff:\n%s",
			key,
			dumpValue(expected),
			dumpValue(o.value[key]),
			diffValues(expected, o.value[key]))
	}
	return o
}

// ValueNotEqual succeeds if object's value for given key is not equal to given value.
// Before comparison, both values are converted to canonical form.
//
// value should map[string]interface{} or struct.
//
// If object doesn't contain any value for given key, failure is reported.
//
// Example:
//  object := NewObject(t, map[string]interface{}{"foo": 123})
//  object.ValueNotEqual("foo", "bad value")  // success
//  object.ValueNotEqual("bar", "bad value")  // failure! (key is missing)
func (o *Object) ValueNotEqual(key string, value interface{}) *Object {
	if !o.containsKey(key) {
		o.chain.fail("\nexpected object containing key '%s', but got:\n%s",
			key, dumpValue(o.value))
		return o
	}
	expected, ok := canonValue(&o.chain, value)
	if !ok {
		return o
	}
	if reflect.DeepEqual(expected, o.value[key]) {
		o.chain.fail("\nexpected value for key '%s' not equal to:\n%s",
			key, dumpValue(expected))
	}
	return o
}

func (o *Object) containsKey(key string) bool {
	for k := range o.value {
		if k == key {
			return true
		}
	}
	return false
}

func (o *Object) containsMap(sm interface{}) bool {
	submap, ok := canonMap(&o.chain, sm)
	if !ok {
		return false
	}
	return checkContainsMap(o.value, submap)
}

func checkContainsMap(outer, inner map[string]interface{}) bool {
	for k, iv := range inner {
		ov, ok := outer[k]
		if !ok {
			return false
		}
		if ovm, ok := ov.(map[string]interface{}); ok {
			if ivm, ok := iv.(map[string]interface{}); ok {
				if !checkContainsMap(ovm, ivm) {
					return false
				}
				continue
			}
		}
		if !reflect.DeepEqual(ov, iv) {
			return false
		}
	}
	return true
}
