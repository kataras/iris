package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/rewrite"
)

func main() {
	app := iris.New()

	/*
		rewriteOptions := rewrite.Options{
			RedirectMatch: []string{
				"301 /seo/(.*) /$1",
				"301 /docs/v12(.*) /docs",
				"301 /old(.*) /",
			}}
		OR Load from file:
	*/
	rewriteOptions := rewrite.LoadOptions("redirects.yml")
	rewriteEngine, err := rewrite.New(rewriteOptions)
	if err != nil { // reports any line parse errors.
		app.Logger().Fatal(err)
	}

	app.Get("/", index)
	app.Get("/about", about)
	app.Get("/docs", docs)

	/*
		// To use it per-party, even if not route match:
		app.UseRouter(rewriteEngine.Handler)
		// To use it per-party when route match:
		app.Use(rewriteEngine.Handler)
		//
		// To use it on a single route just pass it to the Get/Post method.
		// To make the entire application respect the rewrite rules
		// you have to wrap the Iris Router and pass the Wrapper method instead,
		// (recommended way to use this middleware, right before Listen/Run):
	*/
	app.WrapRouter(rewriteEngine.Wrapper)

	// http://localhost:8080/seo
	// http://localhost:8080/about
	// http://localhost:8080/docs/v12/hello
	// http://localhost:8080/docs/v12some
	// http://localhost:8080/oldsome
	// http://localhost:8080/oldindex/random
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
