// The sitemap controller: /sitemap.xml built from the registered static routes when the application is built.
//
// Dynamic, subdomain and offline routes are excluded automatically;
// ExcludeSitemap drops a route explicitly, and SetLastMod, SetChangeFreq and
// SetPriority fill the matching XML elements of a route's <url> entry.
//
// Learn: publish a sitemap that follows your routes, and tune or exclude single entries.
// Before: ../api-graph
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl http://localhost:8080/sitemap.xml   # 200, three <url> entries: /, /about and /docs
//
// Next: ../patches
package main

import (
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/sitemap"
)

func main() {
	app := iris.New()

	// The domain is the absolute prefix of every <loc> in the output. Mount
	// order does not matter: the controller reads the route list on build.
	app.Router("/", sitemap.New("https://www.example.com"))

	app.Get("/", text("index"))

	app.Get("/about", text("about")).
		SetLastMod(time.Now()).
		SetChangeFreq("monthly").
		SetPriority(0.8)

	app.Get("/docs", text("docs")).
		SetChangeFreq("weekly").
		SetPriority(0.5)

	// Excluded: opted out explicitly.
	app.Get("/internal/metrics", text("metrics")).ExcludeSitemap()

	// Excluded: dynamic routes are never listed.
	app.Get("/posts/{id:uint64}", func(ctx iris.Context) {
		ctx.Writef("post %d", ctx.Params().GetUint64("id"))
	})

	app.Listen(":8080")
}

func text(s string) iris.Handler {
	return func(ctx iris.Context) {
		ctx.WriteString(s)
	}
}
