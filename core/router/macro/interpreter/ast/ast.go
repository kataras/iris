package ast

import (
	"reflect"
	"strings"
)

// ParamType holds the necessary information about a parameter type.
type ParamType struct {
	Indent  string   // the name of the parameter type.
	Aliases []string // any aliases, can be empty.

	GoType reflect.Kind // the go type useful for "mvc" and "hero" bindings.

	Default bool // if true then empty type param will target this and its functions will be available to the rest of the param type's funcs.
	End     bool // if true then it should be declared at the end of a route path and can accept any trailing path segment as one parameter.

	invalid bool // only true if returned by the parser via `LookupParamType`.
}

// ParamTypeUnExpected is the unexpected parameter type.
var ParamTypeUnExpected = ParamType{invalid: true}

func (pt ParamType) String() string {
	return pt.Indent
}

// Assignable returns true if the "k" standard type
// is assignabled to this ParamType.
func (pt ParamType) Assignable(k reflect.Kind) bool {
	return pt.GoType == k
}

// GetDefaultParamType accepts a list of ParamType and returns its default.
// If no `Default` specified:
// and len(paramTypes) > 0 then it will return the first one,
// otherwise it returns a "string" parameter type.
func GetDefaultParamType(paramTypes ...ParamType) ParamType {
	for _, pt := range paramTypes {
		if pt.Default == true {
			return pt
		}
	}

	if len(paramTypes) > 0 {
		return paramTypes[0]
	}

	return ParamType{Indent: "string", GoType: reflect.String, Default: true}
}

// ValidKind will return true if at least one param type is supported
// for this std kind.
func ValidKind(k reflect.Kind, paramTypes ...ParamType) bool {
	for _, pt := range paramTypes {
		if pt.GoType == k {
			return true
		}
	}

	return false
}

// LookupParamType accepts the string
// representation of a parameter type.
// Available:
// "string"
// "number" or "int"
// "long" or "int64"
// "uint8"
// "uint64"
// "boolean" or "bool"
// "alphabetical"
// "file"
// "path"
func LookupParamType(indent string, paramTypes ...ParamType) (ParamType, bool) {
	for _, pt := range paramTypes {
		if pt.Indent == indent {
			return pt, true
		}

		for _, alias := range pt.Aliases {
			if alias == indent {
				return pt, true
			}
		}
	}

	return ParamTypeUnExpected, false
}

// LookupParamTypeFromStd accepts the string representation of a standard go type.
// It returns a ParamType, but it may differs for example
// the alphabetical, file, path and string are all string go types, so
// make sure that caller resolves these types before this call.
//
// string matches to string
// int matches to int/number
// int64 matches to int64/long
// uint64 matches to uint64
// bool matches to bool/boolean
func LookupParamTypeFromStd(goType string, paramTypes ...ParamType) (ParamType, bool) {
	goType = strings.ToLower(goType)
	for _, pt := range paramTypes {
		if strings.ToLower(pt.GoType.String()) == goType {
			return pt, true
		}
	}

	return ParamTypeUnExpected, false
}

// ParamStatement is a struct
// which holds all the necessary information about a macro parameter.
// It holds its type (string, int, alphabetical, file, path),
// its source ({param:type}),
// its name ("param"),
// its attached functions by the user (min, max...)
// and the http error code if that parameter
// failed to be evaluated.
type ParamStatement struct {
	Src       string      // the original unparsed source, i.e: {id:int range(1,5) else 404}
	Name      string      // id
	Type      ParamType   // int
	Funcs     []ParamFunc // range
	ErrorCode int         // 404
}

// ParamFunc holds the name of a parameter's function
// and its arguments (values)
// A param func is declared with:
// {param:int range(1,5)},
// the range is the
// param function name
// the 1 and 5 are the two param function arguments
// range(1,5)
type ParamFunc struct {
	Name string   // range
	Args []string // ["1","5"]
}
