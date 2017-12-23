package di

import "reflect"

// Values is a shortcut of []reflect.Value,
// it makes easier to remove and add dependencies.
type Values []reflect.Value

// NewValues returns new empty (dependencies) values.
func NewValues() Values {
	return Values{}
}

// Clone returns a copy of the current values.
func (bv Values) Clone() Values {
	if n := len(bv); n > 0 {
		values := make(Values, n, n)
		copy(values, bv)
		return values
	}

	return NewValues()
}

// CloneWithFieldsOf will return a copy of the current values
// plus the "s" struct's fields that are filled(non-zero) by the caller.
func (bv Values) CloneWithFieldsOf(s interface{}) Values {
	values := bv.Clone()

	// add the manual filled fields to the dependencies.
	filledFieldValues := LookupNonZeroFieldsValues(ValueOf(s), true)
	values = append(values, filledFieldValues...)
	return values
}

// Len returns the length of the current "bv" values slice.
func (bv Values) Len() int {
	return len(bv)
}

// Add adds values as dependencies, if the struct's fields
// or the function's input arguments needs them, they will be defined as
// bindings (at build-time) and they will be used (at serve-time).
func (bv *Values) Add(values ...interface{}) {
	bv.AddValues(ValuesOf(values)...)
}

// AddValues same as `Add` but accepts reflect.Value dependencies instead of interface{}
// and appends them to the list if they pass some checks.
func (bv *Values) AddValues(values ...reflect.Value) {
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
func (bv Values) Has(value interface{}) bool {
	return bv.valueTypeExists(reflect.TypeOf(value))
}

func (bv Values) valueTypeExists(typ reflect.Type) bool {
	for _, in := range bv {
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

	bv.Add(v)
	return true
}
