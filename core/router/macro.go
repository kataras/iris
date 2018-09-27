package router

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/core/router/macro"
	"github.com/kataras/iris/core/router/macro/interpreter/ast"
)

// defaultMacros returns a new macro map which
// contains the default router's named param types functions.
func defaultMacros() *macro.Map {
	macros := macro.NewMap()
	// registers the String and Int default macro funcs
	// user can add or override of his own funcs later on
	// i.e:
	// app.Macro.String.RegisterFunc("equal", func(eqWith string) func(string) bool {
	//	return func(paramValue string) bool {
	//	   return eqWith == paramValue
	//  }})
	registerBuiltinsMacroFuncs(macros)

	return macros
}

func registerBuiltinsMacroFuncs(out *macro.Map) {
	// register the String which is the default type if not
	// parameter type is specified or
	// if a given parameter into path given but the func doesn't exist on the
	// parameter type's function list.
	//
	// these can be overridden by the user, later on.
	registerStringMacroFuncs(out.String)
	registerIntMacroFuncs(out.Int)
	registerIntMacroFuncs(out.Long)
	registerAlphabeticalMacroFuncs(out.Alphabetical)
	registerFileMacroFuncs(out.File)
	registerPathMacroFuncs(out.Path)
}

// String
// anything one part
func registerStringMacroFuncs(out *macro.Macro) {
	// this can be used everywhere, it's to help users to define custom regexp expressions
	// on all macros
	out.RegisterFunc("regexp", func(expr string) macro.EvaluatorFunc {
		regexpEvaluator := macro.MustNewEvaluatorFromRegexp(expr)
		return regexpEvaluator
	})

	// checks if param value starts with the 'prefix' arg
	out.RegisterFunc("prefix", func(prefix string) macro.EvaluatorFunc {
		return func(paramValue string) bool {
			return strings.HasPrefix(paramValue, prefix)
		}
	})

	// checks if param value ends with the 'suffix' arg
	out.RegisterFunc("suffix", func(suffix string) macro.EvaluatorFunc {
		return func(paramValue string) bool {
			return strings.HasSuffix(paramValue, suffix)
		}
	})

	// checks if param value contains the 's' arg
	out.RegisterFunc("contains", func(s string) macro.EvaluatorFunc {
		return func(paramValue string) bool {
			return strings.Contains(paramValue, s)
		}
	})

	// checks if param value's length is at least 'min'
	out.RegisterFunc("min", func(min int) macro.EvaluatorFunc {
		return func(paramValue string) bool {
			return len(paramValue) >= min
		}
	})
	// checks if param value's length is not bigger than 'max'
	out.RegisterFunc("max", func(max int) macro.EvaluatorFunc {
		return func(paramValue string) bool {
			return max >= len(paramValue)
		}
	})
}

// Int
// only numbers (0-9)
func registerIntMacroFuncs(out *macro.Macro) {
	// checks if the param value's int representation is
	// bigger or equal than 'min'
	out.RegisterFunc("min", func(min int) macro.EvaluatorFunc {
		return func(paramValue string) bool {
			n, err := strconv.Atoi(paramValue)
			if err != nil {
				return false
			}
			return n >= min
		}
	})

	// checks if the param value's int representation is
	// smaller or equal than 'max'
	out.RegisterFunc("max", func(max int) macro.EvaluatorFunc {
		return func(paramValue string) bool {
			n, err := strconv.Atoi(paramValue)
			if err != nil {
				return false
			}
			return n <= max
		}
	})

	// checks if the param value's int representation is
	// between min and max, including 'min' and 'max'
	out.RegisterFunc("range", func(min, max int) macro.EvaluatorFunc {
		return func(paramValue string) bool {
			n, err := strconv.Atoi(paramValue)
			if err != nil {
				return false
			}

			if n < min || n > max {
				return false
			}
			return true
		}
	})
}

// Alphabetical
// letters only (upper or lowercase)
func registerAlphabeticalMacroFuncs(out *macro.Macro) {

}

// File
// letters (upper or lowercase)
// numbers (0-9)
// underscore (_)
// dash (-)
// point (.)
// no spaces! or other character
func registerFileMacroFuncs(out *macro.Macro) {

}

// Path
// File+slashes(anywhere)
// should be the latest param, it's the wildcard
func registerPathMacroFuncs(out *macro.Macro) {

}

// compileRoutePathAndHandlers receives a route info and returns its parsed/"compiled" path
// and the new handlers (prepend all the macro's handler, if any).
//
// It's not exported for direct use.
func compileRoutePathAndHandlers(handlers context.Handlers, tmpl *macro.Template) (string, context.Handlers, error) {
	// parse the path to node's path, now.
	path, err := convertTmplToNodePath(tmpl)
	if err != nil {
		return tmpl.Src, handlers, err
	}
	// prepend the macro handler to the route, now,
	// right before the register to the tree, so routerbuilder.UseGlobal will work as expected.
	if len(tmpl.Params) > 0 {
		macroEvaluatorHandler := convertTmplToHandler(tmpl)
		// may return nil if no really need a macro handler evaluator
		if macroEvaluatorHandler != nil {
			handlers = append(context.Handlers{macroEvaluatorHandler}, handlers...)
		}
	}

	return path, handlers, nil
}

func convertTmplToNodePath(tmpl *macro.Template) (string, error) {
	routePath := tmpl.Src
	if len(tmpl.Params) > 0 {
		if routePath[len(routePath)-1] == '/' {
			routePath = routePath[0 : len(routePath)-2] // remove the last "/" if macro syntax instead of underline's
		}
	}

	// if it has started with {} and it's valid
	// then the tmpl.Params will be filled,
	// so no any further check needed
	for i, p := range tmpl.Params {
		if p.Type == ast.ParamTypePath {
			if i != len(tmpl.Params)-1 {
				return "", errors.New("parameter type \"ParamTypePath\" should be putted to the very last of a path")
			}
			routePath = strings.Replace(routePath, p.Src, WildcardParam(p.Name), 1)
		} else {
			routePath = strings.Replace(routePath, p.Src, Param(p.Name), 1)
		}
	}

	return routePath, nil
}

// note: returns nil if not needed, the caller(router) should be check for that before adding that on route's Middleware
func convertTmplToHandler(tmpl *macro.Template) context.Handler {

	needMacroHandler := false

	// check if we have params like: {name:string} or {name} or {anything:path} without else keyword or any functions used inside these params.
	// 1. if we don't have, then we don't need to add a handler before the main route's handler (as I said, no performance if macro is not really used)
	// 2. if we don't have any named params then we don't need a handler too.
	for _, p := range tmpl.Params {
		if len(p.Funcs) == 0 && (p.Type == ast.ParamTypeUnExpected || p.Type == ast.ParamTypeString || p.Type == ast.ParamTypePath) && p.ErrCode == http.StatusNotFound {
		} else {
			// println("we need handler for: " + tmpl.Src)
			needMacroHandler = true
		}
	}

	if !needMacroHandler {
		// println("we don't need handler for: " + tmpl.Src)
		return nil
	}

	return func(tmpl macro.Template) context.Handler {
		return func(ctx context.Context) {
			for _, p := range tmpl.Params {
				paramValue := ctx.Params().Get(p.Name)
				// first, check for type evaluator
				if !p.TypeEvaluator(paramValue) {
					ctx.StatusCode(p.ErrCode)
					ctx.StopExecution()
					return
				}

				// then check for all of its functions
				for _, evalFunc := range p.Funcs {
					if !evalFunc(paramValue) {
						ctx.StatusCode(p.ErrCode)
						ctx.StopExecution()
						return
					}
				}

			}
			// if all passed, just continue
			ctx.Next()
		}
	}(*tmpl)

}
