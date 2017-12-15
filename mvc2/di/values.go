package di

import (
	"reflect"
)

type Values []reflect.Value

func NewValues() Values {
	return Values{}
}

// Add binds values to this controller, if you want to share
// binding values between controllers use the Engine's `Bind` function instead.
func (bv *Values) Add(values ...interface{}) {
	for _, val := range values {
		bv.AddValue(reflect.ValueOf(val))
	}
}

// AddValue same as `Add` but accepts reflect.Value
// instead.
func (bv *Values) AddValue(values ...reflect.Value) {
	for _, v := range values {
		if !goodVal(v) {
			continue
		}
		*bv = append(*bv, v)
	}
}

// Remove unbinds a binding value based on the type,
// it returns true if at least one field is not binded anymore.
//
// The "n" indicates the number of elements to remove, if <=0 then it's 1,
// this is useful because you may have bind more than one value to two or more fields
// with the same type.
func (bv *Values) Remove(value interface{}, n int) bool {
	return bv.remove(reflect.TypeOf(value), n)
}

func (bv *Values) remove(typ reflect.Type, n int) (ok bool) {
	input := *bv
	for i, in := range input {
		if equalTypes(in.Type(), typ) {
			ok = true
			input = input[:i+copy(input[i:], input[i+1:])]
			if n > 1 {
				continue
			}
			break
		}
	}

	*bv = input

	return
}

// Has returns true if a binder responsible to
// bind and return a type of "typ" is already registered to this controller.
func (bv *Values) Has(value interface{}) bool {
	return bv.valueTypeExists(reflect.TypeOf(value))
}

func (bv *Values) valueTypeExists(typ reflect.Type) bool {
	input := *bv
	for _, in := range input {
		if equalTypes(in.Type(), typ) {
			return true
		}
	}
	return false
}

// AddOnce binds a value to the controller's field with the same type,
// if it's not binded already.
//
// Returns false if binded already or the value is not the proper one for binding,
// otherwise true.
func (bv *Values) AddOnce(value interface{}) bool {
	return bv.addIfNotExists(reflect.ValueOf(value))
}

func (bv *Values) addIfNotExists(v reflect.Value) bool {
	var (
		typ = v.Type() // no element, raw things here.
	)

	if !goodVal(v) {
		return false
	}

	if bv.valueTypeExists(typ) {
		return false
	}

	bv.AddValue(v)
	return true
}
