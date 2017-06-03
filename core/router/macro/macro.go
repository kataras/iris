// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package macro

import (
	"fmt"
	"reflect"
	"regexp"
	"unicode"

	"github.com/kataras/iris/core/router/macro/interpreter/ast"
)

// final evaluator signature for both param types and param funcs
type EvaluatorFunc func(paramValue string) bool

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
	Macro struct {
		Evaluator EvaluatorFunc
		funcs     []ParamFunc
	}

	ParamEvaluatorBuilder func([]ast.ParamFuncArg) EvaluatorFunc

	ParamFunc struct {
		Name string
		Func ParamEvaluatorBuilder
	}
)

func newMacro(evaluator EvaluatorFunc) *Macro {
	return &Macro{Evaluator: evaluator}
}

// at boot time, per param
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

type MacroMap struct {
	// string type
	// anything
	String *Macro
	// int type
	// only numbers (0-9)
	Int *Macro
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

func NewMacroMap() *MacroMap {
	return &MacroMap{
		// it allows everything, so no need for a regexp here.
		String:       newMacro(func(string) bool { return true }),
		Int:          newMacro(MustNewEvaluatorFromRegexp("^[0-9]+$")),
		Alphabetical: newMacro(MustNewEvaluatorFromRegexp("^[a-zA-Z ]+$")),
		File:         newMacro(MustNewEvaluatorFromRegexp("^[a-zA-Z0-9_.-]*$")),
		// it allows everything, we have String and Path as different
		// types because I want to give the opportunity to the user
		// to organise the macro functions based on wildcard or single dynamic named path parameter.
		// Should be the last.
		Path: newMacro(func(string) bool { return true }),
	}
}

func (m *MacroMap) Lookup(typ ast.ParamType) *Macro {
	switch typ {
	case ast.ParamTypeInt:
		return m.Int
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
