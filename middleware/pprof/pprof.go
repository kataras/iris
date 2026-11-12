// Package pprof provides native pprof support via middleware.
//
// Its entry point is New(), which returns a *ProfilingRouter rather than a
// Handler: pprof is a set of sibling endpoints (heap, profile, trace, ...), so
// the value is registered as a party with app.Any("/debug/pprof/{action:path}",
// pprof.New()) and it dispatches on the sub-path itself.
//
// Labels policy: Iris sets no pprof labels. If labels are ever added,
// they must carry route metadata only (route path, method), never
// request data — see the SECURITY WARNING on `New` for why.
package pprof

import (
	"html/template"
	"net/http/pprof"
	rpprof "runtime/pprof"
	"sort"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
)

func init() {
	context.SetHandlerName("iris/middleware/pprof.*", "iris.profiling")

	/* for our readers:
	apps.OnApplicationRegistered(func(app *iris.Application) {
		app.Any("/debug/pprof/cmdline", iris.FromStd(pprof.Cmdline))
		app.Any("/debug/pprof/profile", iris.FromStd(pprof.Profile))
		app.Any("/debug/pprof/symbol", iris.FromStd(pprof.Symbol))
		app.Any("/debug/pprof/trace", iris.FromStd(pprof.Trace))

		app.Any("/debug/pprof /debug/pprof/{action:string}", New())
	})
	*/
}

// net/http/pprof copy:
var profileDescriptions = map[string]string{
	"allocs":        "A sampling of all past memory allocations",
	"block":         "Stack traces that led to blocking on synchronization primitives",
	"cmdline":       "The command line invocation of the current program",
	"goroutine":     "Stack traces of all current goroutines",
	"goroutineleak": "Stack traces of goroutines the runtime believes are leaked (durably blocked with no way to be unblocked)",
	"heap":          "A sampling of memory allocations of live objects. You can specify the gc GET parameter to run GC before taking the heap sample.",
	"mutex":         "Stack traces of holders of contended mutexes",
	"profile":       "CPU profile. You can specify the duration in the seconds GET parameter. After you get the profile file, use the go tool pprof command to investigate the profile.",
	"threadcreate":  "Stack traces that led to the creation of new OS threads",
	"trace":         "A trace of execution of the current program. You can specify the duration in the seconds GET parameter. After you get the trace file, use the go tool trace command to investigate the trace.",
}

// ProfilingRouter is the pprof (profile, cmdline, symbol, goroutine, heap, threadcreate, debug/block) Router.
type ProfilingRouter struct{}

// New returns a new pprof (profile, cmdline, symbol, goroutine, heap, threadcreate, debug/block) Router.
//
// SECURITY WARNING: these endpoints MUST NOT be reachable without authentication.
//
//	GET /debug/pprof/goroutine?debug=2
//
// dumps every goroutine's stack - source paths, function names and often
// argument values, which is where session identifiers and tokens live. The heap
// profile discloses the same through allocation sites. Anyone who can reach the
// route gets a detailed picture of the running process.
//
// Goroutine dumps also include the goroutines' pprof labels, and for go1.27
// modules labels print in panic traceback headers too - bypassing the recovery
// middleware's header redaction. Never put request data in pprof labels.
//
// Usage - always behind a guard:
//
//	import "github.com/kataras/iris/v14/middleware/pprof"
//	import "github.com/kataras/iris/v14/middleware/basicauth"
//	[...]
//	app := iris.New()
//
//	auth := basicauth.Default(map[string]string{"admin": os.Getenv("PPROF_PASSWORD")})
//	app.Router("/debug/pprof", pprof.New()).Use(auth)
//
// or restrict it by address with the ipaccess middleware, or bind the profiling
// endpoints to a separate, internal-only listener.
func New() *ProfilingRouter {
	return &ProfilingRouter{}
}

// Routing registers the necessary routes for the pprof middleware.
// Implements the `router.Controller` interface.
func (p *ProfilingRouter) Routing(r router.Router) {
	r.Get("/ /{action:path}", Handler)
}

// Handler is the pprof (profile, cmdline, symbol, goroutine, heap, threadcreate, debug/block) middleware handler.
// Note: Route MUST have the last named parameter wildcard named '{action:path}'.
// Example:
//
//	app.Handle("GET", "/debug/pprof /debug/pprof/{action:path}", pprof.Handler)
func Handler(ctx *context.Context) {
	if action := ctx.Params().GetString("action"); action != "" {
		pprof.Handler(action).ServeHTTP(ctx.ResponseWriter(), ctx.Request())
		return
	}

	ctx.SetHeader("X-Content-Type-Options", "nosniff")
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")

	type profile struct {
		Name  string
		Href  string
		Desc  string
		Count int
	}
	type page struct {
		Path     string
		Profiles []profile
	}

	var profiles []profile
	for _, p := range rpprof.Profiles() {
		profiles = append(profiles, profile{
			Name:  p.Name(),
			Href:  p.Name() + "?debug=1",
			Desc:  profileDescriptions[p.Name()],
			Count: p.Count(),
		})
	}

	// Adding other profiles exposed from within this package
	for _, p := range []string{"cmdline", "profile", "trace"} {
		profiles = append(profiles, profile{
			Name: p,
			Href: p,
			Desc: profileDescriptions[p],
		})
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	if err := indexTmpl.Execute(ctx, page{
		Path:     ctx.Path(),
		Profiles: profiles,
	}); err != nil {
		ctx.Application().Logger().Error(err)
	}
}

var indexTmpl = template.Must(template.New("index").Parse(`<html>
<head>
<title>{{.Path}}</title>
<style>
.profile-name{
	display:inline-block;
	width:6rem;
}
</style>
</head>
<body>
{{.Path}}<br>
<br>
Types of profiles available:
<table>
<thead><td>Count</td><td>Profile</td></thead>
{{ $path := .Path}}
{{range .Profiles}}
	<tr>
	<td>{{.Count}}</td><td><a href={{$path}}/{{.Href}}>{{.Name}}</a></td>
	</tr>
{{end}}
</table>
<a href="{{.Path}}/goroutine?debug=2">full goroutine stack dump</a>
<br/>
<p>
Profile Descriptions:
<ul>
{{range .Profiles}}
<li><div class=profile-name>{{.Name}}:</div> {{.Desc}}</li>
{{end}}
</ul>
</p>
</body>
</html>
`))
