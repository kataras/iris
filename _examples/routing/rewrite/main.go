package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/rewrite"
)

func main() {
	app := iris.New()
	app.Get("/", index)
	app.Get("/about", about)
	app.Get("/docs", docs)

	app.Subdomain("test").Get("/", testIndex)

	redirects := rewrite.Load("redirects.yml")
	app.WrapRouter(redirects)

	// http://mydomain.com:8080/seo/about     -> http://www.mydomain.com:8080/about
	// http://test.mydomain.com:8080
	// http://localhost:8080/seo              -> http://localhost:8080
	// http://localhost:8080/about
	// http://localhost:8080/docs/v12/hello   -> http://localhost:8080/docs
	// http://localhost:8080/docs/v12some     -> http://localhost:8080/docs
	// http://localhost:8080/oldsome          -> http://localhost:8080
	// http://localhost:8080/oldindex/random  -> http://localhost:8080
	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.WriteString("Index")
}

func about(ctx iris.Context) {
	ctx.WriteString("About")
}

func docs(ctx iris.Context) {
	ctx.WriteString("Docs")
}

func testIndex(ctx iris.Context) {
	ctx.WriteString("Test Subdomain Index")
}

/* More...
rewriteOptions := rewrite.Options{
	RedirectMatch: []string{
		"301 /seo/(.*) /$1",
		"301 /docs/v12(.*) /docs",
		"301 /old(.*) /",
	},
	PrimarySubdomain: "www",
}
rewriteEngine, err := rewrite.New(rewriteOptions)

// To use it per-party use its `Handler` method. Even if not route match:
app.UseRouter(rewriteEngine.Handler)
// To use it per-party when route match:
app.Use(rewriteEngine.Handler)
//
// To use it on a single route just pass it to the Get/Post method.
//
// To make the entire application respect the redirect rules
// you have to wrap the Iris Router and pass the `Rewrite` method instead
// as we did at this example.
*/
