package macro

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/kataras/iris/core/router/macro/interpreter/ast"
)

// EvaluatorFunc is the signature for both param types and param funcs.
// It should accepts the param's value as string
// and return true if validated otherwise false.
type EvaluatorFunc func(paramValue string) bool

// NewEvaluatorFromRegexp accepts a regexp "expr" expression
// and returns an EvaluatorFunc based on that regexp.
// the regexp is compiled before return.
//
// Returns a not-nil error on regexp compile failure.
func NewEvaluatorFromRegexp(expr string) (EvaluatorFunc, error) {
	if expr == "" {
		return nil, fmt.Errorf("empty regex expression")
	}

	// add the last $ if missing (and not wildcard(?))
	if i := expr[len(expr)-1]; i != '$' && i != '*' {
		expr += "$"
	}

	r, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}

	return r.MatchString, nil
}

// MustNewEvaluatorFromRegexp same as NewEvaluatorFromRegexp
// but it panics on the "expr" parse failure.
func MustNewEvaluatorFromRegexp(expr string) EvaluatorFunc {
	r, err := NewEvaluatorFromRegexp(expr)
	if err != nil {
		panic(err)
	}
	return r
}

var (
	goodParamFuncReturnType  = reflect.TypeOf(func(string) bool { return false })
	goodParamFuncReturnType2 = reflect.TypeOf(EvaluatorFunc(func(string) bool { return false }))
)

func goodParamFunc(typ reflect.Type) bool {
	// should be a func
	// which returns a func(string) bool
	if typ.Kind() == reflect.Func {
		if typ.NumOut() == 1 {
			typOut := typ.Out(0)
			if typOut == goodParamFuncReturnType || typOut == goodParamFuncReturnType2 {
				return true
			}
		}
	}

	return false
}

// goodParamFuncName reports whether the function name is a valid identifier.
func goodParamFuncName(name string) bool {
	if name == "" {
		return false
	}
	// valid names are only letters and _
	for _, r := range name {
		switch {
		case r == '_':
		case !unicode.IsLetter(r):
			return false
		}
	}
	return true
}

// the convertBuilderFunc return value is generating at boot time.
// convertFunc converts an interface to a valid full param function.
func convertBuilderFunc(fn interface{}) ParamEvaluatorBuilder {

	typFn := reflect.TypeOf(fn)
	if !goodParamFunc(typFn) {
		return nil
	}

	numFields := typFn.NumIn()

	return func(args []string) EvaluatorFunc {
		if len(args) != numFields {
			// no variadics support, for now.
			panic("args should be the same len as numFields")
		}
		var argValues []reflect.Value
		for i := 0; i < numFields; i++ {
			field := typFn.In(i)
			arg := args[i]

			// try to convert the string literal as we get it from the parser.
			var (
				v   interface{}
				err error
			)

			// try to get the value based on the expected type.
			switch field.Kind() {
			case reflect.Int:
				v, err = strconv.Atoi(arg)
			case reflect.Int8:
				v, err = strconv.ParseInt(arg, 10, 8)
			case reflect.Int16:
				v, err = strconv.ParseInt(arg, 10, 16)
			case reflect.Int32:
				v, err = strconv.ParseInt(arg, 10, 32)
			case reflect.Int64:
				v, err = strconv.ParseInt(arg, 10, 64)
			case reflect.Uint8:
				v, err = strconv.ParseUint(arg, 10, 8)
			case reflect.Uint16:
				v, err = strconv.ParseUint(arg, 10, 16)
			case reflect.Uint32:
				v, err = strconv.ParseUint(arg, 10, 32)
			case reflect.Uint64:
				v, err = strconv.ParseUint(arg, 10, 64)
			case reflect.Float32:
				v, err = strconv.ParseFloat(arg, 32)
			case reflect.Float64:
				v, err = strconv.ParseFloat(arg, 64)
			case reflect.Bool:
				v, err = strconv.ParseBool(arg)
			case reflect.Slice:
				if len(arg) > 1 {
					if arg[0] == '[' && arg[len(arg)-1] == ']' {
						// it is a single argument but as slice.
						v = strings.Split(arg[1:len(arg)-1], ",") // only string slices.
					}
				}

			default:
				v = arg
			}

			if err != nil {
				panic(fmt.Sprintf("on field index: %d: %v", i, err))
			}

			argValue := reflect.ValueOf(v)
			if expected, got := field.Kind(), argValue.Kind(); expected != got {
				panic(fmt.Sprintf("fields should have the same type: [%d] expected %s but got %s", i, expected, got))
			}

			argValues = append(argValues, argValue)
		}

		evalFn := reflect.ValueOf(fn).Call(argValues)[0].Interface()

		var evaluator EvaluatorFunc
		// check for typed and not typed
		if _v, ok := evalFn.(EvaluatorFunc); ok {
			evaluator = _v
		} else if _v, ok = evalFn.(func(string) bool); ok {
			evaluator = _v
		}
		return func(paramValue string) bool {
			return evaluator(paramValue)
		}
	}
}

type (
	// Macro represents the parsed macro,
	// which holds
	// the evaluator (param type's evaluator + param functions evaluators)
	// and its param functions.
	//
	// Any type contains its own macro
	// instance, so an String type
	// contains its type evaluator
	// which is the "Evaluator" field
	// and it can register param functions
	// to that macro which maps to a parameter type.
	Macro struct {
		Evaluator EvaluatorFunc
		funcs     []ParamFunc
	}

	// ParamEvaluatorBuilder is a func
	// which accepts a param function's arguments (values)
	// and returns an EvaluatorFunc, its job
	// is to make the macros to be registered
	// by user at the most generic possible way.
	ParamEvaluatorBuilder func([]string) EvaluatorFunc

	// ParamFunc represents the parsed
	// parameter function, it holds
	// the parameter's name
	// and the function which will build
	// the evaluator func.
	ParamFunc struct {
		Name string
		Func ParamEvaluatorBuilder
	}
)

func newMacro(evaluator EvaluatorFunc) *Macro {
	return &Macro{Evaluator: evaluator}
}

// RegisterFunc registers a parameter function
// to that macro.
// Accepts the func name ("range")
// and the function body, which should return an EvaluatorFunc
// a bool (it will be converted to EvaluatorFunc later on),
// i.e RegisterFunc("min", func(minValue int) func(paramValue string) bool){})
func (m *Macro) RegisterFunc(funcName string, fn interface{}) {
	fullFn := convertBuilderFunc(fn)
	m.registerFunc(funcName, fullFn)
}

func (m *Macro) registerFunc(funcName string, fullFn ParamEvaluatorBuilder) {
	if !goodParamFuncName(funcName) {
		return
	}

	for _, fn := range m.funcs {
		if fn.Name == funcName {
			fn.Func = fullFn
			return
		}
	}

	m.funcs = append(m.funcs, ParamFunc{
		Name: funcName,
		Func: fullFn,
	})
}

func (m *Macro) getFunc(funcName string) ParamEvaluatorBuilder {
	for _, fn := range m.funcs {
		if fn.Name == funcName {
			if fn.Func == nil {
				continue
			}
			return fn.Func
		}
	}
	return nil
}

// Map contains the default macros mapped to their types.
// This is the manager which is used by the caller to register custom
// parameter functions per param-type (String, Int, Long, Boolean, Alphabetical, File, Path).
type Map struct {
	// string type
	// anything
	String *Macro

	// int type
	// both positive and negative numbers, any number of digits.
	Number *Macro
	// int64 as int64 type
	// -9223372036854775808 to 9223372036854775807.
	Int64 *Macro
	// uint8 as uint8 type
	// 0 to 255.
	Uint8 *Macro
	// uint64 as uint64 type
	// 0 to 18446744073709551615.
	Uint64 *Macro

	// boolean as bool type
	// a string which is "1" or "t" or "T" or "TRUE" or "true" or "True"
	// or "0" or "f" or "F" or "FALSE" or "false" or "False".
	Boolean *Macro
	// alphabetical/letter type
	// letters only (upper or lowercase)
	Alphabetical *Macro
	// file type
	// letters (upper or lowercase)
	// numbers (0-9)
	// underscore (_)
	// dash (-)
	// point (.)
	// no spaces! or other character
	File *Macro
	// path type
	// anything, should be the last part
	Path *Macro
}

// NewMap returns a new macro Map with default
// type evaluators.
//
// Learn more at:  https://github.com/kataras/iris/tree/master/_examples/routing/dynamic-path
func NewMap() *Map {
	simpleNumberEvalutator := MustNewEvaluatorFromRegexp("^-?[0-9]+$")
	return &Map{
		// it allows everything, so no need for a regexp here.
		String: newMacro(func(string) bool { return true }),
		Number: newMacro(simpleNumberEvalutator), //"^(-?0\\.[0-9]*[1-9]+[0-9]*$)|(^-?[1-9]+[0-9]*((\\.[0-9]*[1-9]+[0-9]*$)|(\\.[0-9]+)))|(^-?[1-9]+[0-9]*$)|(^0$){1}")), //("^-?[0-9]+$")),
		Int64: newMacro(func(paramValue string) bool {
			if !simpleNumberEvalutator(paramValue) {
				return false
			}
			_, err := strconv.ParseInt(paramValue, 10, 64)
			// if err == strconv.ErrRange...
			return err == nil
		}), //("^-[1-9]|-?[1-9][0-9]{1,14}|-?1000000000000000|-?10000000000000000|-?100000000000000000|-?[1-9]000000000000000000|-?9[0-2]00000000000000000|-?92[0-2]0000000000000000|-?922[0-3]000000000000000|-?9223[0-3]00000000000000|-?92233[0-7]0000000000000|-?922337[0-2]000000000000|-?92233720[0-3]0000000000|-?922337203[0-6]000000000|-?9223372036[0-8]00000000|-?92233720368[0-5]0000000|-?922337203685[0-4]000000|-?9223372036854[0-7]00000|-?92233720368547[0-7]0000|-?922337203685477[0-5]000|-?922337203685477[56]000|[0-9]$")),
		Uint8: newMacro(MustNewEvaluatorFromRegexp("^([0-9]|[1-8][0-9]|9[0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$")),
		Uint64: newMacro(func(paramValue string) bool {
			if !simpleNumberEvalutator(paramValue) {
				return false
			}
			_, err := strconv.ParseUint(paramValue, 10, 64)
			return err == nil
		}), //("^[0-9]|[1-9][0-9]{1,14}|1000000000000000|10000000000000000|100000000000000000|1000000000000000000|1[0-8]000000000000000000|18[0-4]00000000000000000|184[0-4]0000000000000000|1844[0-6]000000000000000|18446[0-7]00000000000000|184467[0-4]0000000000000|1844674[0-4]000000000000|184467440[0-7]0000000000|1844674407[0-3]000000000|18446744073[0-7]00000000|1844674407370000000[0-9]|18446744073709[0-5]00000|184467440737095[0-5]0000|1844674407370955[0-2]000$")),
		Boolean: newMacro(func(paramValue string) bool {
			// a simple if statement is faster than regex ^(true|false|True|False|t|0|f|FALSE|TRUE)$
			// in this case.
			_, err := strconv.ParseBool(paramValue)
			return err == nil
		}),
		Alphabetical: newMacro(MustNewEvaluatorFromRegexp("^[a-zA-Z ]+$")),
		File:         newMacro(MustNewEvaluatorFromRegexp("^[a-zA-Z0-9_.-]*$")),
		// it allows everything, we have String and Path as different
		// types because I want to give the opportunity to the user
		// to organise the macro functions based on wildcard or single dynamic named path parameter.
		// Should be the last.
		Path: newMacro(func(string) bool { return true }),
	}
}

// Lookup returns the specific Macro from the map
// based on the parameter type.
// i.e if ast.ParamTypeNumber then it will return the m.Number.
// Returns the m.String if not matched.
func (m *Map) Lookup(typ ast.ParamType) *Macro {
	switch typ {
	case ast.ParamTypeNumber:
		return m.Number
	case ast.ParamTypeInt64:
		return m.Int64
	case ast.ParamTypeUint8:
		return m.Uint8
	case ast.ParamTypeUint64:
		return m.Uint64
	case ast.ParamTypeBoolean:
		return m.Boolean
	case ast.ParamTypeAlphabetical:
		return m.Alphabetical
	case ast.ParamTypeFile:
		return m.File
	case ast.ParamTypePath:
		return m.Path
	default:
		return m.String
	}
}
