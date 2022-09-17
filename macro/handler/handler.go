// Package handler is the highest level module of the macro package which makes use the rest of the macro package,
// it is mainly used, internally, by the router package.
package handler

import (
	"fmt"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/memstore"
	"github.com/kataras/iris/v12/macro"
)

// ParamErrorHandler is a special type of Iris handler which receives
// any error produced by a path type parameter evaluator and let developers
// customize the output instead of the
// provided error code 404 or anyother status code given on the `else` literal.
//
// Note that the builtin macros return error too, but they're handled
// by the `else` literal (error code). To change this behavior
// and send a custom error response you have to register it:
//
//	app.Macros().Get("uuid").HandleError(func(ctx iris.Context, paramIndex int, err error)).
//
// You can also set custom macros by `app.Macros().Register`.
//
// See macro.HandleError to set it.
type ParamErrorHandler = func(*context.Context, int, error) // alias.

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
	for i := range tmpl.Params {
		p := tmpl.Params[i]
		if p.CanEval() {
			// if at least one needs it, then create the handler.
			needsMacroHandler = true

			if p.HandleError != nil {
				// Check for its type.
				if _, ok := p.HandleError.(ParamErrorHandler); !ok {
					panic(fmt.Sprintf("HandleError input argument must be a type of func(iris.Context, int, error) but got: %T", p.HandleError))
				}
			}
			break
		}
	}

	return
}

// MakeHandler creates and returns a handler from a macro template, the handler evaluates each of the parameters if necessary at all.
// If the template does not contain any dynamic attributes and a special handler is NOT required
// then it returns a nil handler.
func MakeHandler(tmpl macro.Template) context.Handler {
	filter := MakeFilter(tmpl)

	return func(ctx *context.Context) {
		if !filter(ctx) {
			if ctx.GetCurrentRoute().StatusErrorCode() == ctx.GetStatusCode() {
				ctx.Next()
			} else {
				ctx.StopExecution()
			}

			return
		}

		// if all passed or the next is the registered error handler to handle this status code,
		// just continue.
		ctx.Next()
	}
}

// MakeFilter returns a Filter which reports whether a specific macro template
// and its parameters pass the serve-time validation.
func MakeFilter(tmpl macro.Template) context.Filter {
	if !CanMakeHandler(tmpl) {
		return nil
	}

	return func(ctx *context.Context) bool {
		for i := range tmpl.Params {
			p := tmpl.Params[i]
			if !p.CanEval() {
				continue // allow.
			}

			// 07-29-2019
			// changed to retrieve by param index in order to support
			// different parameter names for routes with
			// different param types (and probably different param names i.e {name:string}, {id:uint64})
			// in the exact same path pattern.
			//
			// Same parameter names are not allowed, different param types in the same path
			// should have different name e.g. {name} {id:uint64};
			// something like {name} and {name:uint64}
			// is bad API design and we do NOT allow it by-design.
			entry, found := ctx.Params().Store.GetEntryAt(p.Index)
			if !found {
				// should never happen.
				ctx.StatusCode(p.ErrCode) // status code can change from an error handler, set it here.
				return false
			}

			value, passed := p.Eval(entry.String())
			if !passed {
				ctx.StatusCode(p.ErrCode) // status code can change from an error handler, set it here.
				if value != nil && p.HandleError != nil {
					// The "value" is an error here, always (see template.Eval).
					// This is always a type of ParamErrorHandler at this state (see CanMakeHandler).
					p.HandleError.(ParamErrorHandler)(ctx, p.Index, value.(error))
				}
				return false
			}

			// Fixes binding different path parameters names,
			//
			// app.Get("/{fullname:string}", strHandler)
			// app.Get("/{id:int}", idHandler)
			//
			// before that user didn't see anything
			// but under the hoods the set-ed value was a type of string instead of type of int,
			// because store contained both "fullname" (which set-ed by the router itself on its string representation)
			// and "id" by the param evaluator (see core/router/handler.go and bindMultiParamTypesHandler->MakeFilter)
			// and the MVC get by index (e.g. 0) therefore
			// it got the "fullname" of type string instead of "id" int if /{int} requested.
			// which is critical for faster type assertion in the upcoming, new iris dependency injection (20 Feb 2020).
			ctx.Params().Store[p.Index] = memstore.Entry{
				Key:      p.Name,
				ValueRaw: value,
			}

			// for i, v := range ctx.Params().Store {
			// 	fmt.Printf("[%d:%s] macro/handler/handler.go: param passed: %s(%v of type: %T)\n", i, v.Key,
			// 		p.Src, v.ValueRaw, v.ValueRaw)
			// }
		}

		return true
	}
}
