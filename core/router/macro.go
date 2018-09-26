package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router/macro"
	"github.com/kataras/iris/core/router/macro/interpreter/ast"
)

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
		if ast.IsTrailing(p.Type) {
			if i != len(tmpl.Params)-1 {
				return "", fmt.Errorf("parameter type \"%s\" should be putted to the very last of a path", p.Type.Indent())
			}
			routePath = strings.Replace(routePath, p.Src, WildcardParam(p.Name), 1)
		} else {
			routePath = strings.Replace(routePath, p.Src, Param(p.Name), 1)
		}
	}

	return routePath, nil
}

// Note: returns nil if not needed, the caller(router) should check for that before adding that on route's Middleware.
func convertTmplToHandler(tmpl *macro.Template) context.Handler {

	needMacroHandler := false

	// check if we have params like: {name:string} or {name} or {anything:path} without else keyword or any functions used inside these params.
	// 1. if we don't have, then we don't need to add a handler before the main route's handler (as I said, no performance if macro is not really used)
	// 2. if we don't have any named params then we don't need a handler too.
	for _, p := range tmpl.Params {
		if len(p.Funcs) == 0 && (ast.IsMaster(p.Type) || ast.IsTrailing(p.Type)) && p.ErrCode == http.StatusNotFound {
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
