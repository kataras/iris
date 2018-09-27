package macro

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
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

	return func(args []ast.ParamFuncArg) EvaluatorFunc {
		if len(args) != numFields {
			// no variadics support, for now.
			panic("args should be the same len as numFields")
		}
		var argValues []reflect.Value
		for i := 0; i < numFields; i++ {
			field := typFn.In(i)
			arg := args[i]

			if field.Kind() != reflect.TypeOf(arg).Kind() {
				panic("fields should have the same type")
			}

			argValues = append(argValues, reflect.ValueOf(arg))
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
	ParamEvaluatorBuilder func([]ast.ParamFuncArg) EvaluatorFunc

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
	// uint type
	// only positive numbers (+0-9)
	// it could be uint/uint32 but we keep int for simplicity
	Int *Macro
	// long an int64 type
	// only positive numbers (+0-9)
	// it could be uint64 but we keep int64 for simplicity
	Long *Macro
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
	return &Map{
		// it allows everything, so no need for a regexp here.
		String: newMacro(func(string) bool { return true }),
		Int:    newMacro(MustNewEvaluatorFromRegexp("^[0-9]+$")),
		Long:   newMacro(MustNewEvaluatorFromRegexp("^[0-9]+$")),
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
// i.e if ast.ParamTypeInt then it will return the m.Int.
// Returns the m.String if not matched.
func (m *Map) Lookup(typ ast.ParamType) *Macro {
	switch typ {
	case ast.ParamTypeInt:
		return m.Int
	case ast.ParamTypeLong:
		return m.Long
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
