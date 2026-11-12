package reflex

import (
	"reflect"
	"runtime"
)

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
	for i := range n {
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
	for i := range n {
		out := fn.Out(i)
		params = append(params, FuncParam{
			Index: i,
			Type:  out,
		})
	}
	return params
}

// FuncName returns the raw name of a function.
// Use the NameResolver to get the custom name.
func FuncName(v any) string {
	if v == nil {
		return ""
	}

	ptr := ValueOf(v).Pointer()
	funcName, _, _ := FuncPtrNameAndFileLine(ptr)
	return funcName
}

// FuncFileLine returns the func's file and line information.
func FuncFileLine(v any) (file string, line int) {
	if v == nil {
		return
	}

	ptr := ValueOf(v).Pointer()
	_, file, line = FuncPtrNameAndFileLine(ptr)
	return
}

// FuncPtrNameAndFileLine returns the function name, the filename and the line
// of the function at the given program counter.
//
// Display-only fidelity: since Go 1.27 two distinct code-identical function
// literals may share a code pointer, in which case the reported name, file and
// line belong to whichever literal the compiler kept. Never use these values
// as a function identity; they are for tracing and debug output.
func FuncPtrNameAndFileLine(ptr uintptr) (funcName, filename string, line int) {
	pc := runtime.FuncForPC(ptr)
	funcName = pc.Name()
	filename, line = pc.FileLine(pc.Entry())
	return
}
