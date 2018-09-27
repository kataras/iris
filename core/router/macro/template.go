package macro

import (
	"reflect"

	"github.com/kataras/iris/core/router/macro/interpreter/ast"
	"github.com/kataras/iris/core/router/macro/interpreter/parser"
)

// Template contains a route's path full parsed template.
//
// Fields:
// Src is the raw source of the path, i.e /users/{id:int min(1)}
// Params is the list of the Params that are being used to the
// path,  i.e the min as param name and  1 as the param argument.
type Template struct {
	// Src is the original template given by the client
	Src    string          `json:"src"`
	Params []TemplateParam `json:"params"`
}

// TemplateParam is the parsed macro parameter's template
// they are being used to describe the param's syntax result.
type TemplateParam struct {
	Src string `json:"src"` // the unparsed param'false source
	// Type is not useful anywhere here but maybe
	// it's useful on host to decide how to convert the path template to specific router's syntax
	Type          ast.ParamType   `json:"type"`
	Name          string          `json:"name"`
	Index         int             `json:"index"`
	ErrCode       int             `json:"errCode"`
	TypeEvaluator ParamEvaluator  `json:"-"`
	Funcs         []reflect.Value `json:"-"`
}

// Parse takes a full route path and a macro map (macro map contains the macro types with their registered param functions)
// and returns a new Template.
// It builds all the parameter functions for that template
// and their evaluators, it's the api call that makes use the interpeter's parser -> lexer.
func Parse(src string, macros Macros) (*Template, error) {
	types := make([]ast.ParamType, len(macros))
	for i, m := range macros {
		types[i] = m
	}

	params, err := parser.Parse(src, types)
	if err != nil {
		return nil, err
	}
	t := new(Template)
	t.Src = src

	for idx, p := range params {
		funcMap := macros.Lookup(p.Type)
		typEval := funcMap.Evaluator

		tmplParam := TemplateParam{
			Src:           p.Src,
			Type:          p.Type,
			Name:          p.Name,
			Index:         idx,
			ErrCode:       p.ErrorCode,
			TypeEvaluator: typEval,
		}

		for _, paramfn := range p.Funcs {
			tmplFn := funcMap.getFunc(paramfn.Name)
			if tmplFn == nil { // if not find on this type, check for Master's which is for global funcs too.
				if m := macros.GetMaster(); m != nil {
					tmplFn = m.getFunc(paramfn.Name)
				}

				if tmplFn == nil { // if not found then just skip this param.
					continue
				}
			}

			evalFn := tmplFn(paramfn.Args)
			if evalFn.IsNil() || !evalFn.IsValid() || evalFn.Kind() != reflect.Func {
				continue
			}
			tmplParam.Funcs = append(tmplParam.Funcs, evalFn)
		}

		t.Params = append(t.Params, tmplParam)
	}

	return t, nil
}
