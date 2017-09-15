package methodfunc

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/kataras/iris/context"
)

var posWords = map[int]string{
	0: "",
	1: "first",
	2: "second",
	3: "third",
	4: "forth",
	5: "five",
	6: "sixth",
	7: "seventh",
	8: "eighth",
	9: "ninth",
}

func genParamKey(argIdx int) string {
	return "param" + posWords[argIdx] // paramfirst, paramsecond...
}

const (
	paramTypeInt     = "int"
	paramTypeLong    = "long"
	paramTypeBoolean = "boolean"
	paramTypeString  = "string"
	paramTypePath    = "path"
)

var macroTypes = map[string]string{
	"int":    paramTypeInt,
	"int64":  paramTypeLong,
	"bool":   paramTypeBoolean,
	"string": paramTypeString,
	// there is "path" param type but it's being captured "on-air"
	// "file" param type is not supported by the current implementation, yet
	// but if someone ask for it I'll implement it, it's easy.
}

type funcParser struct {
	info  FuncInfo
	lexer *lexer
}

func newFuncParser(info FuncInfo) *funcParser {
	return &funcParser{
		info:  info,
		lexer: newLexer(info.Trailing),
	}
}

func (p *funcParser) parse() (*ast, error) {
	a := new(ast)
	funcArgPos := 0

	for {
		w := p.lexer.next()
		if w == "" {
			break
		}

		if w == tokenBy {
			typ := p.info.Type
			funcArgPos++ // starting with 1 because in typ.NumIn() the first is the struct receiver.

			if p.lexer.peekPrev() == tokenBy || typ.NumIn() == 1 { // ByBy, then act this second By like a path
				a.relPath += "/" + strings.ToLower(w)
				continue
			}

			if typ.NumIn() <= funcArgPos {
				return nil, errors.New("keyword 'By' found but length of input receivers are not match for " +
					p.info.Name)
			}

			var (
				paramKey  = genParamKey(funcArgPos) // paramfirst, paramsecond...
				paramType = paramTypeString         // default string
			)

			// string, int...
			goType := typ.In(funcArgPos).Name()

			if p.lexer.peekNext() == tokenWildcard {
				p.lexer.skip() // skip the Wildcard word.
				paramType = paramTypePath
			} else if pType, ok := macroTypes[goType]; ok {
				// it's not wildcard, so check base on our available macro types.
				paramType = pType
			} else {
				return nil, errors.New("invalid syntax for " + p.info.Name)
			}

			a.paramKeys = append(a.paramKeys, paramKey)
			a.paramTypes = append(a.paramTypes, paramType)
			// /{paramfirst:path}, /{paramfirst:long}...
			a.relPath += fmt.Sprintf("/{%s:%s}", paramKey, paramType)
			a.dynamic = true
			continue
		}

		a.relPath += "/" + strings.ToLower(w)
	}
	return a, nil
}

type ast struct {
	paramKeys  []string // paramfirst, paramsecond... [0]
	paramTypes []string // string, int, long, path... [0]
	relPath    string
	dynamic    bool // when paramKeys (and paramTypes, are equal) > 0
}

// moved to func_caller#buildMethodcall, it's bigger and with repeated code
// than this, below function but it's faster.
// func (a *ast) MethodCall(ctx context.Context, f reflect.Value) {
// 	if a.dynamic {
// 		f.Call(a.paramValues(ctx))
// 		return
// 	}
//
// 	f.Interface().(func())()
// }

func (a *ast) paramValues(ctx context.Context) []reflect.Value {
	l := len(a.paramKeys)
	values := make([]reflect.Value, l, l)
	for i := 0; i < l; i++ {
		paramKey := a.paramKeys[i]
		paramType := a.paramTypes[i]
		values[i] = getParamValueFromType(ctx, paramType, paramKey)
	}

	return values
}

func getParamValueFromType(ctx context.Context, paramType string, paramKey string) reflect.Value {
	if paramType == paramTypeInt {
		v, _ := ctx.Params().GetInt(paramKey)
		return reflect.ValueOf(v)
	}

	if paramType == paramTypeLong {
		v, _ := ctx.Params().GetInt64(paramKey)
		return reflect.ValueOf(v)
	}

	if paramType == paramTypeBoolean {
		v, _ := ctx.Params().GetBool(paramKey)
		return reflect.ValueOf(v)
	}

	// string, path...
	return reflect.ValueOf(ctx.Params().Get(paramKey))
}
