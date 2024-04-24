package reflex

import "reflect"

// IsFunc reports whether the "kindable" is a type of function.
func IsFunc(typ interface{ Kind() reflect.Kind }) bool {
	return typ.Kind() == reflect.Func
}

// FuncParam holds the properties of function input or output.
type FuncParam struct {
	Index int
	Type  reflect.Type
}

// LookupInputs returns the index and type of each function's input argument.
// Panics if "fn" is not a type of Func.
func LookupInputs(fn reflect.Type) []FuncParam {
	n := fn.NumIn()
	params := make([]FuncParam, 0, n)
	for i := 0; i < n; i++ {
		in := fn.In(i)
		params = append(params, FuncParam{
			Index: i,
			Type:  in,
		})
	}
	return params
}

// LookupOutputs returns the index and type of each function's output argument.
// Panics if "fn" is not a type of Func.
func LookupOutputs(fn reflect.Type) []FuncParam {
	n := fn.NumOut()
	params := make([]FuncParam, 0, n)
	for i := 0; i < n; i++ {
		out := fn.Out(i)
		params = append(params, FuncParam{
			Index: i,
			Type:  out,
		})
	}
	return params
}
