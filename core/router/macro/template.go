// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package macro

import (
	"github.com/kataras/iris/core/router/macro/interpreter/ast"
	"github.com/kataras/iris/core/router/macro/interpreter/parser"
)

type Template struct {
	// Src is the original template given by the client
	Src    string
	Params []TemplateParam
}

type TemplateParam struct {
	Src string // the unparsed param'false source
	// Type is not useful anywhere here but maybe
	// it's useful on host to decide how to convert the path template to specific router's syntax
	Type          ast.ParamType
	Name          string
	ErrCode       int
	TypeEvaluator EvaluatorFunc
	Funcs         []EvaluatorFunc
}

func Parse(src string, macros *MacroMap) (*Template, error) {
	params, err := parser.Parse(src)
	if err != nil {
		return nil, err
	}
	t := new(Template)
	t.Src = src

	for _, p := range params {
		funcMap := macros.Lookup(p.Type)
		typEval := funcMap.Evaluator

		tmplParam := TemplateParam{
			Src:           p.Src,
			Type:          p.Type,
			Name:          p.Name,
			ErrCode:       p.ErrorCode,
			TypeEvaluator: typEval,
		}
		for _, paramfn := range p.Funcs {
			tmplFn := funcMap.getFunc(paramfn.Name)
			if tmplFn == nil { // if not find on this type, check for String's which is for global funcs too
				tmplFn = macros.String.getFunc(paramfn.Name)
				if tmplFn == nil { // if not found then just skip this param
					continue
				}
			}
			evalFn := tmplFn(paramfn.Args)
			if evalFn == nil {
				continue
			}
			tmplParam.Funcs = append(tmplParam.Funcs, evalFn)
		}

		t.Params = append(t.Params, tmplParam)
	}

	return t, nil
}
