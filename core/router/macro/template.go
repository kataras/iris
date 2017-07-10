package macro

import (
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
	Src    string
	Params []TemplateParam
}

// TemplateParam is the parsed macro parameter's template
// they are being used to describe the param's syntax result.
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

// Parse takes a full route path and a macro map (macro map contains the macro types with their registered param functions)
// and returns a new Template.
// It builds all the parameter functions for that template
// and their evaluators, it's the api call that makes use the interpeter's parser -> lexer.
func Parse(src string, macros *Map) (*Template, error) {
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
