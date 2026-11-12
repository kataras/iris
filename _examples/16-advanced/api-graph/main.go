// The apigraph controller: an interactive D3.js map of every registered route, mounted behind an admin guard.
//
// The graph shows paths, methods, handler names and editor links to each
// handler's source line. It is generated after the application builds, so the
// mount order relative to the routes it draws does not matter.
//
// Mount it behind an admin guard, and preferably in development builds only: it
// exposes the whole API surface plus local source file paths. The guard below
// compares the X-Admin-Token request header against the ADMIN_TOKEN
// environment variable ("dev" when unset).
//
// Learn: draw your route table as a page you can click through, and keep it behind a token.
// Before: ../multi-app
//
// Run:
//
//	go run main.go
//
// The page loads d3.js from the network, so open it in a browser with the
// token header set, or read the HTML with curl.
//
// Try:
//
//	curl -i http://localhost:8080/admin/apigraph                       # 401, no token
//	curl -H "X-Admin-Token: dev" http://localhost:8080/admin/apigraph   # 200, the graph page
//
// Next: ../sitemap
package main

import (
	"os"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/apigraph"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	// Some routes to draw.
	app.Get("/", text("index"))
	users := app.Router("/users")
	users.Get("/", text("list users"))
	users.Post("/", text("create user"))
	users.Get("/{id:uint64}", text("get user"))
	users.Put("/{id:uint64}", text("update user"))
	users.Get("/{id:uint64}/posts", text("list user posts"))

	// Themes: "light" (default) or "dark"; register your own with
	// apigraph.RegisterTheme.
	admin := app.Router("/admin", requireAdminToken(adminToken()))
	admin.Router("/apigraph", apigraph.New(apigraph.Options{Theme: "dark"}))

	app.Listen(":8080")
}

func adminToken() string {
	if token := os.Getenv("ADMIN_TOKEN"); token != "" {
		return token
	}

	return "dev"
}

func requireAdminToken(token string) iris.Handler {
	return func(ctx iris.Context) {
		if ctx.Header("X-Admin-Token") != token {
			// The error code carries the 401 and stops the chain; the
			// renderer negotiates JSON or an HTML page from Accept.
			errors.Unauthenticated.Message(ctx, "missing or invalid X-Admin-Token header")
			return
		}

		ctx.Next()
	}
}

func text(s string) iris.Handler {
	return func(ctx iris.Context) {
		ctx.WriteString(s)
	}
}
