// Package pprof provides native pprof support via middleware. See _examples/miscellaneous/pprof
package pprof

import (
	"net/http/pprof"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/handlerconv"
)

// New returns a new pprof (profile, cmdline, symbol, goroutine, heap, threadcreate, debug/block) Middleware.
// Note: Route MUST have the last named parameter wildcard named '{action:path}'
func New() context.Handler {
	indexHandler := handlerconv.FromStd(pprof.Index)
	cmdlineHandler := handlerconv.FromStd(pprof.Cmdline)
	profileHandler := handlerconv.FromStd(pprof.Profile)
	symbolHandler := handlerconv.FromStd(pprof.Symbol)
	goroutineHandler := handlerconv.FromStd(pprof.Handler("goroutine"))
	heapHandler := handlerconv.FromStd(pprof.Handler("heap"))
	threadcreateHandler := handlerconv.FromStd(pprof.Handler("threadcreate"))
	debugBlockHandler := handlerconv.FromStd(pprof.Handler("block"))

	return func(ctx context.Context) {
		ctx.ContentType("text/html")
		actionPathParameter := ctx.Values().GetString("action")
		if len(actionPathParameter) > 1 {
			if strings.Contains(actionPathParameter, "cmdline") {
				cmdlineHandler((ctx))
			} else if strings.Contains(actionPathParameter, "profile") {
				profileHandler(ctx)
			} else if strings.Contains(actionPathParameter, "symbol") {
				symbolHandler(ctx)
			} else if strings.Contains(actionPathParameter, "goroutine") {
				goroutineHandler(ctx)
			} else if strings.Contains(actionPathParameter, "heap") {
				heapHandler(ctx)
			} else if strings.Contains(actionPathParameter, "threadcreate") {
				threadcreateHandler(ctx)
			} else if strings.Contains(actionPathParameter, "debug/block") {
				debugBlockHandler(ctx)
			}
		} else {
			indexHandler(ctx)
		}
	}
}
