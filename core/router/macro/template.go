package macro

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/kataras/iris/v14/core/router/internal/parser"
	"github.com/kataras/iris/v14/core/router/macro/interpreter/ast"
)

// Template contains a route's path full parsed template.
//
// Fields:
// Src is the raw source of the path, i.e /users/{id:int min(1)}
// Params is the list of the parsed parameters of the path, i.e the id as
// param name and int as its type, while the min(1) function is compiled
// into that param's Funcs.
type Template struct {
	// Src is the original template given by the client
	Src    string          `json:"src"`
	Params []TemplateParam `json:"params"`
	// the sanitized text of the template, used for storing
	// a Src safely in text format without special characters.
	SanitizedText string `json:"sanitized_text"`
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
	HandleError any `json:"-"` /* It's not an typed value because of import-cycle,
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
// It returns the type-based value and true, if the "paramValue" is the correct
// type of the registered parameter type and all functions, if any, are passed.
// Otherwise it returns false and either nil or, when the parameter has a
// registered `HandleError` and the type evaluator produced an error value,
// that error itself, so the caller can report it (see `handler.MakeFilter`).
//
// It is called from the converted macro handler (middleware)
// from the higher-level component of "kataras/iris/macro/handler#MakeHandler".
func (p *TemplateParam) Eval(paramValue string) (any, bool) {
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
		// fmt.Printf("macro/template.go: failed to evaluate the parameter: %s with value: %s, returned param value: %v, error handler: %#+v\n", p.Src, paramValue, newValue, p.HandleError)
		if newValue != nil && p.HandleError != nil { // return this error only when a HandleError was registered.
			if _, ok := newValue.(errorInterface); ok {
				// fmt.Printf("macro/template.go: HandleError for param: %s with value: %s\n", p.Src, paramValue)
				return newValue, false // this is an error, see `HandleError` and `MakeFilter`.
			}

			// fmt.Printf("New value: (%T) %v is not an error, but it should be, check the macro's evaluator\n", newValue, newValue)
		}

		return nil, false
	}

	if len(p.Funcs) > 0 {
		paramIn := []reflect.Value{reflect.ValueOf(newValue)}
		for _, evalFunc := range p.Funcs {
			// or make it as func(any) bool and pass directly the "newValue"
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

	tmpl := Template{Src: src, SanitizedText: sanitizeTemplateSourceText(src)}
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
			HandleError:   m.errorHandler, // NOTE: this is the field, not the method! And that's why the SetErrorHandler MUST be called before any routes even registered.
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
			tmplParam.Funcs = append(tmplParam.Funcs, adaptFuncInput(evalFn, m.GoType()))
		}

		tmpl.Params = append(tmpl.Params, tmplParam.preComputed())
	}

	return tmpl, nil
}

var (
	stringType = reflect.TypeFor[string]()
	anyType    = reflect.TypeFor[any]()
)

// adaptFuncInput makes a string func usable on a typed macro.
//
// The master's funcs (prefix, suffix, contains, min, max, eq, eqor, regexp)
// are inherited by every macro that lacks the func, and TemplateParam.Eval
// calls them by reflection with the value the macro's evaluator stored: an int
// for "{id:int prefix(4)}", a uuid.UUID for "{id:uuid prefix(a6)}". Calling a
// func(string) bool with such a value panics inside reflect on the first
// request, so the func is wrapped to receive the value's text instead.
// A func whose input already matches, or a macro that stores strings, is
// returned as is.
func adaptFuncInput(fn reflect.Value, valueType reflect.Type) reflect.Value {
	if valueType == nil || valueType == stringType {
		return fn
	}

	typ := fn.Type()
	if typ.NumIn() != 1 || typ.In(0) != stringType {
		return fn
	}

	outs := make([]reflect.Type, typ.NumOut())
	for i := range outs {
		outs[i] = typ.Out(i)
	}

	return reflect.MakeFunc(reflect.FuncOf([]reflect.Type{anyType}, outs, false),
		func(args []reflect.Value) []reflect.Value {
			text := reflect.ValueOf(fmt.Sprint(args[0].Interface()))
			return fn.Call([]reflect.Value{text})
		})
}

// CountParams returns the length of the dynamic path's input parameters.
func CountParams(macros Macros, fullpath string) int {
	tmpl, _ := Parse(fullpath, macros)
	return len(tmpl.Params)
}

// ContainsParam reports whether the given fullpath contains the "paramToSearch" parameter.
func ContainsParam(macros Macros, fullpath string, paramToSearch string) bool {
	tmpl, _ := Parse(fullpath, macros)
	for _, p := range tmpl.Params {
		if p.Name == paramToSearch {
			return true
		}
	}

	return false
}

var (
	// matches { ... } including the braces.
	sanitizeIdentifierRegexp = regexp.MustCompile(`\{[^}]*\}`)
	// regexp which replaces [, ], {, }, (, ), *, :, and all whitespace with _.
	sanitizeInnerRegexp = regexp.MustCompile(`[{}\[\]()*:\s]`)
)

// /users/{id:int min(1)} -> /users/_id_int_min_1__
// while keeping the static path as it's.
func sanitizeTemplateSourceText(tmplSrc string) string {
	return sanitizeIdentifierRegexp.ReplaceAllStringFunc(tmplSrc, func(braced string) string {
		return sanitizeInnerRegexp.ReplaceAllString(braced, "_")
	})
}
