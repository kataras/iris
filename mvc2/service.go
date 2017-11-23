package mvc2

import (
	"reflect"
)

// // Service is a `reflect.Value` value.
// // We keep that type here,
// // if we ever need to change this type we will not have
// // to refactor the whole mvc's codebase.
// type Service struct {
// 	reflect.Value
// 	typ reflect.Type
// }

// // Valid checks if the service's Value's Value is valid for set or get.
// func (s Service) Valid() bool {
// 	return goodVal(s.Value)
// }

// // Equal returns if the
// func (s Service) Equal(other Service) bool {
// 	return equalTypes(s.typ, other.typ)
// }

// func (s Service) String() string {
// 	return s.Type().String()
// }

// func wrapService(service interface{}) Service {
// 	if s, ok := service.(Service); ok {
// 		return s // if it's a Service already.
// 	}
// 	return Service{
// 		Value: reflect.ValueOf(service),
// 		typ:   reflect.TypeOf(service),
// 	}
// }

// // WrapServices wrap a generic services into structured Service slice.
// func WrapServices(services ...interface{}) []Service {
// 	if l := len(services); l > 0 {
// 		out := make([]Service, l, l)
// 		for i, s := range services {
// 			out[i] = wrapService(s)
// 		}
// 		return out
// 	}
// 	return nil
// }

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
		BindType: typ,
		BindFunc: func(_ []reflect.Value) reflect.Value {
			return val
		},
	}, nil
}
