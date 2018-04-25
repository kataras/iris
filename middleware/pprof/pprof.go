// Package pprof provides native pprof support via middleware. See _examples/miscellaneous/pprof
package pprof

import (
	"html/template"
	"net/http/pprof"
	rpprof "runtime/pprof"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/handlerconv"
)

// New returns a new pprof (profile, cmdline, symbol, goroutine, heap, threadcreate, debug/block) Middleware.
// Note: Route MUST have the last named parameter wildcard named '{action:path}'
func New() context.Handler {
	cmdlineHandler := handlerconv.FromStd(pprof.Cmdline)
	profileHandler := handlerconv.FromStd(pprof.Profile)
	symbolHandler := handlerconv.FromStd(pprof.Symbol)
	goroutineHandler := handlerconv.FromStd(pprof.Handler("goroutine"))
	heapHandler := handlerconv.FromStd(pprof.Handler("heap"))
	threadcreateHandler := handlerconv.FromStd(pprof.Handler("threadcreate"))
	debugBlockHandler := handlerconv.FromStd(pprof.Handler("block"))

	return func(ctx context.Context) {
		ctx.ContentType("text/html")
		action := ctx.Params().Get("action")
		if action != "" {
			if strings.Contains(action, "cmdline") {
				cmdlineHandler((ctx))
			} else if strings.Contains(action, "profile") {
				profileHandler(ctx)
			} else if strings.Contains(action, "symbol") {
				symbolHandler(ctx)
			} else if strings.Contains(action, "goroutine") {
				goroutineHandler(ctx)
			} else if strings.Contains(action, "heap") {
				heapHandler(ctx)
			} else if strings.Contains(action, "threadcreate") {
				threadcreateHandler(ctx)
			} else if strings.Contains(action, "debug/block") {
				debugBlockHandler(ctx)
			}
			return
		}

		profiles := rpprof.Profiles()
		data := map[string]interface{}{
			"Profiles": profiles,
			"Path":     ctx.RequestPath(false),
		}

		if err := indexTmpl.Execute(ctx, data); err != nil {
			ctx.Application().Logger().Error(err)
		}
	}
}

var indexTmpl = template.Must(template.New("index").Parse(`<html>
	<head>
	<title>/{{.Path}}</title>
	</head>
	<body>
	{{.Path}}<br>
	<br>
	profiles:<br>
	<table>
	{{$path := .Path}}
	{{range .Profiles}}
	<tr><td align=right>{{.Count}}<td><a href="{{$path}}/{{.Name}}?debug=1">{{.Name}}</a>
	{{end}}
	</table>
	<br>
	<a href="{{$path}}/goroutine?debug=2">full goroutine stack dump</a><br>
	</body>
	</html>
	`))
