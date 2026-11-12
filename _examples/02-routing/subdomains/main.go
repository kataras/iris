// Subdomain routing: a router bound to a host label instead of a path prefix.
//
// Subdomain(label) returns a child router bound to a host label instead of a
// path prefix; calling it on a child prepends the label, so
// app.Subdomain("admin").Subdomain("panel") registers "panel.admin.".
// WildcardSubdomain() serves every subdomain no static one claimed and belongs
// to the root router only.
//
// The Host requirement: a route with a subdomain needs the application's root
// host at build time. Without iris.Options{Host: ...} the build fails with
// "root host is required when at least one route contains a subdomain".
// Include the port unless it is 80 or 443.
//
// Matching compares the request's Host header against "label." as a prefix, so
// "admin.example.com" matches the admin router while "badmin.example.com" does
// not; "staff.admin.example.com" does not either. Register a nested Subdomain
// for that. Requests whose host is a loopback address (127.0.0.1:8080) are
// never treated as subdomains.
//
// This example uses example.com so that ctx.Subdomain() has something to
// report: it derives the label through the public suffix list, which has no
// entry for "localhost", so a "shop.localhost" request would return "".
// Nothing leaves the machine. curl still talks to localhost and only sets the
// Host header.
//
// Learn: serve a different route table per subdomain with Subdomain and
// WildcardSubdomain, and set the Host option the build needs.
// Before: ../groups
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -H "Host: example.com:8080"             http://localhost:8080/   # 200, root domain: example.com:8080
//	curl -H "Host: admin.example.com:8080"       http://localhost:8080/   # 200, admin dashboard
//	curl -H "Host: panel.admin.example.com:8080" http://localhost:8080/   # 200, admin panel
//	curl -H "Host: shop.example.com:8080"        http://localhost:8080/   # 200, tenant: shop
//	curl -H "Host: badmin.example.com:8080"      http://localhost:8080/   # 200, tenant: badmin (the wildcard, not admin)
//
// Next: ../custom-macro
package main

import (
	"github.com/kataras/iris/v14"
)

func main() {
	// Host is what makes the subdomain routes buildable and matchable.
	app := iris.New(iris.Options{Host: "example.com:8080"})

	// example.com:8080/
	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("root domain: %s", ctx.Host())
	})

	// admin.example.com:8080/ - its own route table, isolated from the root's.
	admin := app.Subdomain("admin")
	admin.Get("/", func(ctx iris.Context) {
		ctx.Writef("admin dashboard")
	})

	// Nested: panel.admin.example.com:8080/
	panel := admin.Subdomain("panel")
	panel.Get("/", func(ctx iris.Context) {
		ctx.Writef("admin panel")
	})

	// Every other subdomain, e.g. shop.example.com:8080/
	wildcard := app.WildcardSubdomain()
	wildcard.Get("/", func(ctx iris.Context) {
		ctx.Writef("tenant: %s", ctx.Subdomain())
	})

	app.Listen(":8080")
}
