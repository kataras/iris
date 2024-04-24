package macro

import (
	"reflect"

	"github.com/kataras/iris/v12/macro/interpreter/ast"
	"github.com/kataras/iris/v12/macro/interpreter/parser"
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

// IsTrailing reports whether this Template is a traling one.
func (t *Template) IsTrailing() bool {
	return len(t.Params) > 0 && ast.IsTrailing(t.Params[len(t.Params)-1].Type)
}

// TemplateParam is the parsed macro parameter's template
// they are being used to describe the param's syntax result.
type TemplateParam struct {
	macro *Macro // keep for reference.

	Src string `json:"src"` // the unparsed param'false source
	// Type is not useful anywhere here but maybe
	// it's useful on host to decide how to convert the path template to specific router's syntax
	Type    ast.ParamType `json:"type"`
	Name    string        `json:"name"`
	Index   int           `json:"index"`
	ErrCode int           `json:"errCode"`
	// Note that, the value MUST BE a type of `handler.ParamErrorHandler`.
	HandleError interface{} `json:"-"` /* It's not an typed value because of import-cycle,
	// neither a special struct required, see `handler.MakeFilter`. */
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
	p.canEval = p.TypeEvaluator != nil || len(p.Funcs) > 0 || p.ErrCode != parser.DefaultParamErrorCode || p.HandleError != nil

	return p
}

// CanEval returns true if this "p" TemplateParam should be evaluated in serve time.
// It is computed before server ran and it is used to determinate if a route needs to build a macro handler (middleware).
func (p *TemplateParam) CanEval() bool {
	return p.canEval
}

type errorInterface interface {
	Error() string
}

// Eval is the most critical part of the TemplateParam.
// It is responsible to return the type-based value if passed otherwise nil.
// If the "paramValue" is the correct type of the registered parameter type
// and all functions, if any, are passed.
//
// It is called from the converted macro handler (middleware)
// from the higher-level component of "kataras/iris/macro/handler#MakeHandler".
func (p *TemplateParam) Eval(paramValue string) (interface{}, bool) {
	if p.TypeEvaluator == nil {
		for _, fn := range p.stringInFuncs {
			if !fn(paramValue) {
				return nil, false
			}
		}
		return paramValue, true
	}

	// fmt.Printf("macro/template.go#L88: Eval for param value: %s and p.Src: %s\n", paramValue, p.Src)

	newValue, passed := p.TypeEvaluator(paramValue)
	if !passed {
		if newValue != nil && p.HandleError != nil { // return this error only when a HandleError was registered.
			if _, ok := newValue.(errorInterface); ok {
				return newValue, false // this is an error, see `HandleError` and `MakeFilter`.
			}
		}

		return nil, false
	}

	if len(p.Funcs) > 0 {
		paramIn := []reflect.Value{reflect.ValueOf(newValue)}
		for _, evalFunc := range p.Funcs {
			// or make it as func(interface{}) bool and pass directly the "newValue"
			// but that would not be as easy for end-developer, so keep that "slower":
			if !evalFunc.Call(paramIn)[0].Interface().(bool) { // i.e func(paramValue int) bool
				return nil, false
			}
		}
	}

	// fmt.Printf("macro/template.go: passed with value: %v and type: %T\n", newValue, newValue)

	return newValue, true
}

// IsMacro reports whether this TemplateParam's underline macro matches the given one.
func (p *TemplateParam) IsMacro(macro *Macro) bool {
	return p.macro == macro
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
			macro: m,

			Src:           p.Src,
			Type:          p.Type,
			Name:          p.Name,
			Index:         idx,
			ErrCode:       p.ErrorCode,
			HandleError:   m.handleError,
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

// CountParams returns the length of the dynamic path's input parameters.
func CountParams(fullpath string, macros Macros) int {
	tmpl, _ := Parse(fullpath, macros)
	return len(tmpl.Params)
}
