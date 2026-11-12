// Writing your own middleware: a timing factory and two guards, each registered
// where its scope belongs.
//
// A middleware is an ordinary iris.Handler: it does its work and either calls
// ctx.Next() to continue the chain or writes a response and returns. There is
// no middleware interface to implement. This example builds a configurable
// timing middleware (a factory closure) and two guards written as methods of
// an API struct, then registers each one where its scope belongs.
//
// A guard that refuses a request renders it through rest/errors: the error
// code carries the status and stops the chain, so the guard has no status
// code to write and no ctx.StopExecution to remember.
//
// The registration family, and what each call covers:
//
//	app.UseRouter(h)     every request, BEFORE routing - matched routes, 404s
//	                     and 405s alike. The place for access logs, request
//	                     IDs, CORS, compression and rate limits.
//	app.Use(h)           (root) every matched route, PLUS the 404 and 405
//	                     chains. app.UseRelevant(h) is the matched-only
//	                     variant; app.UseOnce(h) replaces a handler of the
//	                     same name instead of stacking a second copy.
//	r.Use(h)             (child router) only the routes registered under that
//	                     child - a child's Use delegates to UseRelevant.
//	r.Get(path, g, h)    positional middleware: guards that one route only.
//
// Ordering: layers run outermost-first, and within a layer registration order
// wins. Code placed after ctx.Next() runs on the way out, in reverse. A
// route's chain is frozen when the route is registered, so always register
// middleware before routes.
//
// Learn: write a middleware, choose between UseRouter, Use and a positional
// guard, and stop a request through rest/errors.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i http://localhost:8080/                        # 200, X-Layer: root
//	curl -i http://localhost:8080/api/ping                # 200, X-Layer: root, api
//	curl -i http://localhost:8080/nothing-here            # 404, still timed
//	curl -i http://localhost:8080/admin/settings          # 401, group guard
//	curl -i -H "X-Token: s3cret" http://localhost:8080/admin/settings   # 200
//	curl -i -X PUT -H "X-Token: s3cret" \
//	    http://localhost:8080/admin/settings              # 403, route guard
//	curl -i -X PUT -H "X-Token: s3cret" -H "X-Scopes: read write" \
//	    http://localhost:8080/admin/settings              # 200
//
// Next: ../conditional
package main

import (
	"strings"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	// Before routing: this one also times the requests that end as a 404.
	app.UseRouter(timing(500 * time.Millisecond))

	// Root Use: every matched route plus the 404/405 chains.
	app.Use(layer("root"))

	app.Get("/", func(ctx iris.Context) {
		ctx.Text("index")
	})

	// Child Use: only the routes registered under this child router.
	api := app.Router("/api")
	api.Use(layer("api"))
	api.Get("/ping", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"pong": true})
	})

	// A controller keeps its guards next to the handlers they protect; both
	// read the same struct fields (in a real service: injected dependencies).
	app.Router("/admin", &AdminAPI{token: "s3cret"})

	app.Listen(":8080")
}

// timing is a middleware factory: the returned closure captures "threshold".
// Work placed after ctx.Next() runs once the handlers below have finished,
// which is why response headers must be set BEFORE Next - by then the body,
// and with it the headers, is already on the wire. After Next, log and record
// metrics only.
func timing(threshold time.Duration) iris.Handler {
	return func(ctx iris.Context) {
		start := time.Now()

		ctx.Next()

		if d := time.Since(start); d > threshold {
			ctx.Application().Logger().Warnf("slow request: %s %s took %s",
				ctx.Method(), ctx.Path(), d)
		}
	}
}

// layer stamps the scope it was registered at. AddHeader appends (SetHeader
// would replace), so a route under /api reports one X-Layer value per layer
// it passed through.
func layer(name string) iris.Handler {
	return func(ctx iris.Context) {
		ctx.AddHeader("X-Layer", name)
		ctx.Next()
	}
}

// AdminAPI is a controller: a struct with a Routing method, registered with
// app.Router("/admin", api). Its guards are plain methods, so they share the
// struct's state with the handlers.
type AdminAPI struct {
	token string
}

// Routing is called once, at startup, with the child router of "/admin".
func (api *AdminAPI) Routing(r iris.Router) {
	// Group guard: every admin route needs the token.
	r.Use(api.requireToken)

	r.Get("/settings", api.get)
	// Positional guard: writes additionally need the "write" scope.
	r.Put("/settings", api.requireWriteScope, api.update)
}

func (api *AdminAPI) requireToken(ctx iris.Context) {
	if ctx.Header("X-Token") != api.token {
		errors.Unauthenticated.Message(ctx, "missing or invalid X-Token header")
		return // no Next: the chain ends here.
	}

	ctx.Next()
}

func (api *AdminAPI) requireWriteScope(ctx iris.Context) {
	if !strings.Contains(ctx.Header("X-Scopes"), "write") {
		errors.PermissionDenied.Message(ctx, "the write scope is required")
		return
	}

	ctx.Next()
}

func (api *AdminAPI) get(ctx iris.Context) {
	ctx.JSON(iris.Map{"theme": "dark", "page_size": 25})
}

func (api *AdminAPI) update(ctx iris.Context) {
	ctx.JSON(iris.Map{"updated": true})
}
