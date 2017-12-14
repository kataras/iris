package di

import (
	"reflect"
)

type Values []reflect.Value

func NewValues() Values {
	return Values{}
}

// Bind binds values to this controller, if you want to share
// binding values between controllers use the Engine's `Bind` function instead.
func (bv *Values) Bind(values ...interface{}) {
	for _, val := range values {
		bv.bind(reflect.ValueOf(val))
	}
}

// Add same as `Bind` but accepts reflect.Value
// instead.
func (bv *Values) Add(values ...reflect.Value) {
	for _, v := range values {
		bv.bind(v)
	}
}

func (bv *Values) bind(v reflect.Value) {
	if !goodVal(v) {
		return
	}

	*bv = append(*bv, v)
}

// Unbind unbinds a binding value based on the type,
// it returns true if at least one field is not binded anymore.
//
// The "n" indicates the number of elements to remove, if <=0 then it's 1,
// this is useful because you may have bind more than one value to two or more fields
// with the same type.
func (bv *Values) Unbind(value interface{}, n int) bool {
	return bv.unbind(reflect.TypeOf(value), n)
}

func (bv *Values) unbind(typ reflect.Type, n int) (ok bool) {
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

// BindExists returns true if a binder responsible to
// bind and return a type of "typ" is already registered to this controller.
func (bv *Values) BindExists(value interface{}) bool {
	return bv.bindTypeExists(reflect.TypeOf(value))
}

func (bv *Values) bindTypeExists(typ reflect.Type) bool {
	input := *bv
	for _, in := range input {
		if equalTypes(in.Type(), typ) {
			return true
		}
	}
	return false
}

// BindIfNotExists bind a value to the controller's field with the same type,
// if it's not binded already.
//
// Returns false if binded already or the value is not the proper one for binding,
// otherwise true.
func (bv *Values) BindIfNotExists(value interface{}) bool {
	return bv.bindIfNotExists(reflect.ValueOf(value))
}

func (bv *Values) bindIfNotExists(v reflect.Value) bool {
	var (
		typ = v.Type() // no element, raw things here.
	)

	if !goodVal(v) {
		return false
	}

	if bv.bindTypeExists(typ) {
		return false
	}

	bv.bind(v)
	return true
}
