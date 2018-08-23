package ast

import (
	"reflect"
)

// ParamType is a specific uint8 type
// which holds the parameter types' type.
type ParamType uint8

const (
	// ParamTypeUnExpected is an unexpected parameter type.
	ParamTypeUnExpected ParamType = iota
	// ParamTypeString 	is the string type.
	// If parameter type is missing then it defaults to String type.
	// Allows anything
	// Declaration: /mypath/{myparam:string} or {myparam}
	ParamTypeString

	// ParamTypeNumber is the integer, a number type.
	// Allows both positive and negative numbers, any number of digits.
	// Declaration: /mypath/{myparam:number} or {myparam:int} for backwards-compatibility
	ParamTypeNumber

	// ParamTypeInt64 is a number type.
	// Allows only -9223372036854775808 to 9223372036854775807.
	// Declaration: /mypath/{myparam:int64} or {myparam:long}
	ParamTypeInt64
	// ParamTypeUint8 a number type.
	// Allows only 0 to 255.
	// Declaration: /mypath/{myparam:uint8}
	ParamTypeUint8
	// ParamTypeUint64 a number type.
	// Allows only 0 to 18446744073709551615.
	// Declaration: /mypath/{myparam:uint64}
	ParamTypeUint64

	// ParamTypeBoolean is the bool type.
	// Allows only "1" or "t" or "T" or "TRUE" or "true" or "True"
	// or "0" or "f" or "F" or "FALSE" or "false" or "False".
	// Declaration: /mypath/{myparam:bool} or {myparam:boolean}
	ParamTypeBoolean
	// ParamTypeAlphabetical is the  alphabetical/letter type type.
	// Allows letters only (upper or lowercase)
	// Declaration:  /mypath/{myparam:alphabetical}
	ParamTypeAlphabetical
	// ParamTypeFile  is the file single path type.
	// Allows:
	// letters (upper or lowercase)
	// numbers (0-9)
	// underscore (_)
	// dash (-)
	// point (.)
	// no spaces! or other character
	// Declaration: /mypath/{myparam:file}
	ParamTypeFile
	// ParamTypePath is the multi path (or wildcard) type.
	// Allows anything, should be the last part
	// Declaration: /mypath/{myparam:path}
	ParamTypePath
)

func (pt ParamType) String() string {
	for k, v := range paramTypes {
		if v == pt {
			return k
		}
	}

	return "unexpected"
}

// Not because for a single reason
// a string may be a
// ParamTypeString or a ParamTypeFile
// or a ParamTypePath or ParamTypeAlphabetical.
//
// func ParamTypeFromStd(k reflect.Kind) ParamType {

// Kind returns the std kind of this param type.
func (pt ParamType) Kind() reflect.Kind {
	switch pt {
	case ParamTypeAlphabetical:
		fallthrough
	case ParamTypeFile:
		fallthrough
	case ParamTypePath:
		fallthrough
	case ParamTypeString:
		return reflect.String
	case ParamTypeNumber:
		return reflect.Int
	case ParamTypeInt64:
		return reflect.Int64
	case ParamTypeUint8:
		return reflect.Uint8
	case ParamTypeUint64:
		return reflect.Uint64
	case ParamTypeBoolean:
		return reflect.Bool
	}
	return reflect.Invalid // 0
}

// ValidKind will return true if at least one param type is supported
// for this std kind.
func ValidKind(k reflect.Kind) bool {
	switch k {
	case reflect.String:
		fallthrough
	case reflect.Int:
		fallthrough
	case reflect.Int64:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint64:
		fallthrough
	case reflect.Bool:
		return true
	default:
		return false
	}
}

// Assignable returns true if the "k" standard type
// is assignabled to this ParamType.
func (pt ParamType) Assignable(k reflect.Kind) bool {
	return pt.Kind() == k
}

var paramTypes = map[string]ParamType{
	"string": ParamTypeString,

	"number": ParamTypeNumber,
	"int":    ParamTypeNumber, // same as number.
	"long":   ParamTypeInt64,
	"int64":  ParamTypeInt64, // same as long.
	"uint8":  ParamTypeUint8,
	"uint64": ParamTypeUint64,

	"boolean": ParamTypeBoolean,
	"bool":    ParamTypeBoolean, // same as boolean.

	"alphabetical": ParamTypeAlphabetical,
	"file":         ParamTypeFile,
	"path":         ParamTypePath,
	// could be named also:
	// "tail":
	// "wild"
	// "wildcard"

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
func LookupParamType(ident string) ParamType {
	if typ, ok := paramTypes[ident]; ok {
		return typ
	}
	return ParamTypeUnExpected
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
func LookupParamTypeFromStd(goType string) ParamType {
	switch goType {
	case "string":
		return ParamTypeString
	case "int":
		return ParamTypeNumber
	case "int64":
		return ParamTypeInt64
	case "uint8":
		return ParamTypeUint8
	case "uint64":
		return ParamTypeUint64
	case "bool":
		return ParamTypeBoolean
	default:
		return ParamTypeUnExpected
	}
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
