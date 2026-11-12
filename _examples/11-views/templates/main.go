// HTML pages from templates, with a shared layout.
//
// Views in Iris v14 are a middleware, not a method on the Context. The engine
// is github.com/kataras/blocks: html/template plus layouts and partials.
// blocks.New accepts a directory or an fs.FS, so the templates here are
// embedded with //go:embed and the program runs from anywhere; pages end in
// .html and layouts live in views/layouts. app.UseRouter(view.Handler(templates))
// stores the engine on every request. A handler then either renders with
// view.Render(ctx, page, layout, data) or declares the whole response with
// view.WithTemplate and has no body of its own. A layout prints the page where
// it says {{ template "content" . }}.
//
// On the builder the registration line is RouterMiddlewares(view.Handler(templates)).
//
// Learn: render HTML pages from embedded templates through a layout, pass data
// to them, and register the view engine once for the whole application.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl localhost:8080/            # 200, the index page inside the main layout
//	curl localhost:8080/about       # 200, declared with view.WithTemplate
//	curl localhost:8080/plain       # 200, the same page with no layout
//
// Next: ../../12-database/postgres-mapping
package main

import (
	"embed"
	"log"
	"time"

	"github.com/kataras/blocks"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/view"
	"github.com/kataras/iris/v14/rest/errors"
)

// views/index.html, views/about.html and views/layouts/main.html.
//
//go:embed views
var viewsFS embed.FS

func main() {
	// RootDir points inside the embedded tree; blocks.New("./views") would
	// read the same files from disk and Reload(true) would re-read them on
	// every request during development.
	templates := blocks.New(viewsFS).RootDir("views")
	if err := templates.Load(); err != nil {
		log.Fatal(err)
	}

	app := iris.New()

	// Registered before routing, so every handler on every router can render.
	app.UseRouter(view.Handler(templates))

	app.Get("/", index)
	app.Get("/plain", plain)

	// The declarative form: no handler body at all.
	app.Get("/about", view.WithTemplate(&view.Template{
		Name:   "about",
		Layout: "main",
		Data:   iris.Map{"Title": "About the Bookshelf"},
	}))

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	data := iris.Map{
		"Title":   "Bookshelf",
		"Now":     time.Now().Format(time.RFC1123),
		"Shelves": []string{"sci-fi", "history", "archive"},
	}

	// Page "index" inside layout "main" (views/layouts/main.html).
	if err := view.Render(ctx, "index", "main", data); err != nil {
		errors.Internal.LogErr(ctx, err) // a generic 500; the template error is logged, not sent.
	}
}

func plain(ctx iris.Context) {
	// An empty layout name renders the page on its own.
	if err := view.Render(ctx, "index", "", iris.Map{"Title": "Bookshelf", "Shelves": []string{"sci-fi"}}); err != nil {
		errors.Internal.LogErr(ctx, err)
	}
}
