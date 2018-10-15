package macro

import (
	"reflect"

	"github.com/kataras/iris/core/memstore"
	"github.com/kataras/iris/macro/interpreter/ast"
	"github.com/kataras/iris/macro/interpreter/parser"
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

	stringInFuncs []func(string) bool
	canEval       bool
}

func (p TemplateParam) preComputed() TemplateParam {
	for _, pfn := range p.Funcs {
		if fn, ok := pfn.Interface().(func(string) bool); ok {
			p.stringInFuncs = append(p.stringInFuncs, fn)
		}
	}

	// if true then it should be execute the type parameter or its functions
	// else it can be ignored,
	// i.e {myparam} or {myparam:string} or {myparam:path} ->
	// their type evaluator is nil because they don't do any checks and they don't change
	// the default parameter value's type (string) so no need for any work).
	p.canEval = p.TypeEvaluator != nil || len(p.Funcs) > 0 || p.ErrCode != parser.DefaultParamErrorCode

	return p
}

// CanEval returns true if this "p" TemplateParam should be evaluated in serve time.
// It is computed before server ran and it is used to determinate if a route needs to build a macro handler (middleware).
func (p *TemplateParam) CanEval() bool {
	return p.canEval
}

// Eval is the most critical part of the TEmplateParam.
// It is responsible to return "passed:true" or "not passed:false"
// if the "paramValue" is the correct type of the registered parameter type
// and all functions, if any, are passed.
// "paramChanger" is the same form of context's Params().Set
// we could accept a memstore.Store or even context.RequestParams
// but this form has been chosed in order to test easier and fully decoupled from a request when necessary.
//
// It is called from the converted macro handler (middleware)
// from the higher-level component of "kataras/iris/macro/handler#MakeHandler".
func (p *TemplateParam) Eval(paramValue string, paramSetter memstore.ValueSetter) bool {
	if p.TypeEvaluator == nil {
		for _, fn := range p.stringInFuncs {
			if !fn(paramValue) {
				return false
			}
		}
		return true
	}

	newValue, passed := p.TypeEvaluator(paramValue)
	if !passed {
		return false
	}

	if len(p.Funcs) > 0 {
		paramIn := []reflect.Value{reflect.ValueOf(newValue)}
		for _, evalFunc := range p.Funcs {
			// or make it as func(interface{}) bool and pass directly the "newValue"
			// but that would not be as easy for end-developer, so keep that "slower":
			if !evalFunc.Call(paramIn)[0].Interface().(bool) { // i.e func(paramValue int) bool
				return false
			}
		}
	}

	paramSetter.Set(p.Name, newValue)
	return true
}

// Parse takes a full route path and a macro map (macro map contains the macro types with their registered param functions)
// and returns a new Template.
// It builds all the parameter functions for that template
// and their evaluators, it's the api call that makes use the interpeter's parser -> lexer.
func Parse(src string, macros Macros) (Template, error) {
	types := make([]ast.ParamType, len(macros))
	for i, m := range macros {
		types[i] = m
	}

	tmpl := Template{Src: src}
	params, err := parser.Parse(src, types)
	if err != nil {
		return tmpl, err
	}

	for idx, p := range params {
		m := macros.Lookup(p.Type)
		typEval := m.Evaluator

		tmplParam := TemplateParam{
			Src:           p.Src,
			Type:          p.Type,
			Name:          p.Name,
			Index:         idx,
			ErrCode:       p.ErrorCode,
			TypeEvaluator: typEval,
		}

		for _, paramfn := range p.Funcs {
			tmplFn := m.getFunc(paramfn.Name)
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

		tmpl.Params = append(tmpl.Params, tmplParam.preComputed())
	}

	return tmpl, nil
}
