package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/cache"
	"github.com/kataras/iris/v12/middleware/basicauth"
)

var markdownContents = []byte(`## Hello Markdown

This is a sample of Markdown contents

Features
--------

All features of Sundown are supported, including:

*   **Compatibility**. The Markdown v1.0.3 test suite passes with
    the --tidy option.  Without --tidy, the differences are
    mostly in whitespace and entity escaping, where blackfriday is
    more consistent and cleaner.
`)

// Cache should not be used on handlers that contain dynamic data.
// Cache is a good and a must-feature on static content, i.e "about page" or for a whole blog site.
func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")
	app.Get("/", cache.Handler(10*time.Second), writeMarkdown)
	// To customize the cache handler:
	// cache.Cache(nil).MaxAge(func(ctx iris.Context) time.Duration {
	// 	return time.Duration(ctx.MaxAge()) * time.Second
	// }).AddRule(...).Store(...)
	// saves its content on the first request and serves it instead of re-calculating the content.
	// After 10 seconds it will be cleared and reset.

	pages := app.Party("/pages")
	pages.Use(cache.Handler(10 * time.Second)) // Per Party.
	pages.Get("/", pagesIndex)
	pages.Post("/", pagesIndexPost)

	// Note: on authenticated requests
	// the cache middleare does not run at all (see iris/cache/ruleset).
	auth := basicauth.Default(map[string]string{
		"admin": "admin",
	})
	app.Get("/protected", auth, cache.Handler(5*time.Second), protected)

	// Set custom cache key/identifier,
	// for the sake of the example
	// we will SHARE the keys on both GET and POST routes
	// so the first one is executed that's the body
	// for both of the routes. Please don't do that
	// on production, this is just an example.
	custom := app.Party("/custom")
	custom.Use(cache.WithKey("shared"))
	custom.Use(cache.Handler(10 * time.Second))
	custom.Get("/", customIndex)
	custom.Post("/", customIndexPost)

	app.Listen(":8080")
}

func writeMarkdown(ctx iris.Context) {
	// tap multiple times the browser's refresh button and you will
	// see this println only once every 10 seconds.
	println("Handler executed. Content refreshed.")

	ctx.Markdown(markdownContents)
}

func pagesIndex(ctx iris.Context) {
	println("Handler executed. Content refreshed.")
	ctx.WriteString("GET: hello")
}

func pagesIndexPost(ctx iris.Context) {
	println("Handler executed. Content refreshed.")
	ctx.WriteString("POST: hello")
}

func protected(ctx iris.Context) {
	username, _, _ := ctx.Request().BasicAuth()
	ctx.Writef("Hello, %s!", username)
}

func customIndex(ctx iris.Context) {
	ctx.WriteString("Contents from GET custom index")
}

func customIndexPost(ctx iris.Context) {
	ctx.WriteString("Contents from POST custom index")
}

/* Note that `HandleDir` does use the browser's disk caching by-default
therefore, register the cache handler AFTER any HandleDir calls,
for a faster solution that server doesn't need to keep track of the response
navigate to https://github.com/kataras/iris/blob/main/_examples/cache/client-side/main.go.

The `HandleDir` has its own cache mechanism, read the 'file-server' examples. */
