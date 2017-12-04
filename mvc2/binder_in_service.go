package mvc2

import (
	"reflect"
)

type serviceFieldBinder struct {
	Index  []int
	Binder *InputBinder
}

func getServicesBinderForStruct(binders []*InputBinder, typ reflect.Type) func(elem reflect.Value) {
	fields := lookupFields(typ, -1)
	var validBinders []*serviceFieldBinder

	for _, b := range binders {
		for _, f := range fields {
			if b.BinderType != serviceType {
				continue
			}
			if equalTypes(b.BindType, f.Type) {
				validBinders = append(validBinders,
					&serviceFieldBinder{Index: f.Index, Binder: b})
			}
		}

	}

	if len(validBinders) == 0 {
		return func(_ reflect.Value) {}
	}

	return func(elem reflect.Value) {
		for _, b := range validBinders {
			elem.FieldByIndex(b.Index).Set(b.Binder.BindFunc(nil))
		}
	}
}

// MustMakeServiceInputBinder calls the `MakeServiceInputBinder` and returns its first result, see its docs.
// It panics on error.
func MustMakeServiceInputBinder(service interface{}) *InputBinder {
	s, err := MakeServiceInputBinder(service)
	if err != nil {
		panic(err)
	}
	return s
}

// MakeServiceInputBinder uses a difference/or strange approach,
// we make the services as bind functions
// in order to keep the rest of the code simpler, however we have
// a performance penalty when calling the function instead
// of just put the responsible service to the certain handler's input argument.
func MakeServiceInputBinder(service interface{}) (*InputBinder, error) {
	if service == nil {
		return nil, errNil
	}

	var (
		val = reflect.ValueOf(service)
		typ = val.Type()
	)

	if !goodVal(val) {
		return nil, errBad
	}

	if indirectTyp(typ).Kind() != reflect.Struct {
		// if the pointer's struct is not a struct then return err bad.
		return nil, errBad
	}

	return &InputBinder{
		BinderType: serviceType,
		BindType:   typ,
		BindFunc: func(_ []reflect.Value) reflect.Value {
			return val
		},
	}, nil
}
