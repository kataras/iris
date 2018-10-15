// Package handler is the highest level module of the macro package which makes use the rest of the macro package,
// it is mainly used, internally, by the router package.
package handler

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/macro"
)

// CanMakeHandler reports whether a macro template needs a special macro's evaluator handler to be validated
// before procceed to the next handler(s).
// If the template does not contain any dynamic attributes and a special handler is NOT required
// then it returns false.
func CanMakeHandler(tmpl macro.Template) (needsMacroHandler bool) {
	if len(tmpl.Params) == 0 {
		return
	}

	// check if we have params like: {name:string} or {name} or {anything:path} without else keyword or any functions used inside these params.
	// 1. if we don't have, then we don't need to add a handler before the main route's handler (as I said, no performance if macro is not really used)
	// 2. if we don't have any named params then we don't need a handler too.
	for _, p := range tmpl.Params {
		if p.CanEval() {
			// if at least one needs it, then create the handler.
			needsMacroHandler = true
			break
		}
	}

	return
}

// MakeHandler creates and returns a handler from a macro template, the handler evaluates each of the parameters if necessary at all.
// If the template does not contain any dynamic attributes and a special handler is NOT required
// then it returns a nil handler.
func MakeHandler(tmpl macro.Template) context.Handler {
	if !CanMakeHandler(tmpl) {
		return nil
	}

	return func(ctx context.Context) {
		for _, p := range tmpl.Params {
			if !p.CanEval() {
				continue // allow.
			}

			if !p.Eval(ctx.Params().Get(p.Name), &ctx.Params().Store) {
				ctx.StatusCode(p.ErrCode)
				ctx.StopExecution()
				return
			}
		}
		// if all passed, just continue.
		ctx.Next()
	}
}
