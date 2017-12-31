package ast

import (
	"fmt"
	"reflect"
	"strconv"
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
	// Declaration: /mypath/{myparam:string} or /mypath{myparam}
	ParamTypeString
	// ParamTypeInt is the integer, a number type.
	// Allows only positive numbers (0-9)
	// Declaration: /mypath/{myparam:int}
	ParamTypeInt
	// ParamTypeLong is the integer, a number type.
	// Allows only positive numbers (0-9)
	// Declaration: /mypath/{myparam:long}
	ParamTypeLong
	// ParamTypeBoolean is the bool type.
	// Allows only "1" or "t" or "T" or "TRUE" or "true" or "True"
	// or "0" or "f" or "F" or "FALSE" or "false" or "False".
	// Declaration: /mypath/{myparam:boolean}
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
	case ParamTypeInt:
		return reflect.Int
	case ParamTypeLong:
		return reflect.Int64
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
	"string":       ParamTypeString,
	"int":          ParamTypeInt,
	"long":         ParamTypeLong,
	"boolean":      ParamTypeBoolean,
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
// "int"
// "long"
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
// int matches to int
// int64 matches to long
// bool matches to boolean
func LookupParamTypeFromStd(goType string) ParamType {
	switch goType {
	case "string":
		return ParamTypeString
	case "int":
		return ParamTypeInt
	case "int64":
		return ParamTypeLong
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

// ParamFuncArg represents a single parameter function's argument
type ParamFuncArg interface{}

// ParamFuncArgToInt converts and returns
// any type of "a", to an integer.
func ParamFuncArgToInt(a ParamFuncArg) (int, error) {
	switch a.(type) {
	case int:
		return a.(int), nil
	case string:
		return strconv.Atoi(a.(string))
	case int64:
		return int(a.(int64)), nil
	default:
		return -1, fmt.Errorf("unexpected function argument type: %q", a)
	}
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
	Name string         // range
	Args []ParamFuncArg // [1,5]
}
