// Wrapping the router: raw net/http code that runs before any route matching.
//
// A router wrapper is raw net/http code that runs before routing, for every
// request, with no iris.Context involved yet:
//
//	func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)
//
// Call next to continue into the router (or the next wrapper), skip it to
// short-circuit. Because it sees the request before any matching happens, it is
// the right place for cross-cutting URL rewrites and net/http-level tooling.
//
// Three registration methods, differing only in ordering:
//
//   - AddRouterWrapper(fn)     defers wrapping until the router builds;
//     the first registered runs first.
//   - PrependRouterWrapper(fn) like AddRouterWrapper but always first;
//     for path-normalization concerns.
//   - WrapRouter(fn)           wraps immediately, so it ends up inside the
//     deferred ones: it runs last, right before the router.
//
// Reach for UseRouter instead when the work needs context features; it is Iris
// middleware and still runs before route matching.
//
// Learn: rewrite or short-circuit a request before routing with WrapRouter,
// AddRouterWrapper and PrependRouterWrapper, and know which one runs first.
// Before: ../custom-macro
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i http://localhost:8080/v2/users                     # 200, X-Routed-Path: /v2/users
//	curl -i http://localhost:8080/legacy/users                 # 200, rewritten, no redirect: X-Rewritten-From: /legacy/users
//	curl -i http://localhost:8080/legacy/users/                # 200, trailing slash normalized
//	curl -i "http://localhost:8080/v2/users?maintenance=1"     # 503, router skipped, Retry-After: 120
//
// Next: ../../03-controllers/basic
package main

import (
	"net/http"
	"strings"

	"github.com/kataras/iris/v14"
)

func main() {
	app := iris.New()

	// Runs first, whatever the registration order: normalize the path so
	// "/v2/users/" reaches "/v2/users" without a path-correction redirect.
	app.PrependRouterWrapper(func(w http.ResponseWriter, r *http.Request,
		next http.HandlerFunc) {
		if p := strings.TrimSuffix(r.URL.Path, "/"); p != "" {
			r.URL.Path = p
		}

		next(w, r)
	})

	// First AddRouterWrapper, so it runs before the next one: a kill switch.
	// next is never called, so the router does not even acquire a context.
	app.AddRouterWrapper(func(w http.ResponseWriter, r *http.Request,
		next http.HandlerFunc) {
		if r.URL.Query().Get("maintenance") != "" {
			w.Header().Set("Retry-After", "120")
			http.Error(w, "under maintenance", http.StatusServiceUnavailable)
			return
		}

		next(w, r)
	})

	// Second AddRouterWrapper: serve the old /legacy/* URLs from the /v2 routes,
	// without redirects and without touching the routes themselves.
	app.AddRouterWrapper(func(w http.ResponseWriter, r *http.Request,
		next http.HandlerFunc) {
		if p, ok := strings.CutPrefix(r.URL.Path, "/legacy/"); ok {
			w.Header().Set("X-Rewritten-From", r.URL.Path)
			r.URL.Path = "/v2/" + p
		}

		next(w, r)
	})

	// Registered with WrapRouter, so it is the innermost wrapper: by now every
	// rewrite above has been applied and this is the path the router will match.
	app.WrapRouter(func(w http.ResponseWriter, r *http.Request,
		next http.HandlerFunc) {
		w.Header().Set("X-Routed-Path", r.URL.Path)
		next(w, r)
	})

	// One route table, reachable through /v2 and through /legacy.
	app.Router("/v2", func(r iris.Router) {
		r.Get("/users", func(ctx iris.Context) {
			ctx.JSON(iris.Map{"version": 2, "path": ctx.Path()})
		})
	})

	app.Listen(":8080")
}
