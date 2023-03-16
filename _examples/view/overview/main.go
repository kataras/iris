package main

import (
	"html/template"
	"time"

	"github.com/kataras/iris/v12"
)

// ViewFunctions presents some builtin functions
// for html view engines. See `View.Funcs` or `view/html.Funcs` and etc.
var Functions = template.FuncMap{
	"Now": time.Now,
}

func main() {
	app := iris.New()

	// with default template funcs:
	//
	// - {{ urlpath "mynamedroute" "pathParameter_ifneeded" }}
	// - {{ render "header.html" . }}
	// - {{ render_r "header.html" . }} // partial relative path to current page
	// - {{ yield . }}
	// - {{ current . }}
	app.RegisterView(iris.HTML("./templates", ".html").
		Funcs(Functions). // Optionally register some more builtin functions.
		Reload(false))    // Set Reload to true on development.

	app.Get("/", func(ctx iris.Context) {
		// enable compression based on Accept-Encoding (e.g. "gzip"),
		// alternatively: app.Use(iris.Compression).
		ctx.CompressWriter(true)
		// the .Name inside the ./templates/hi.html.
		ctx.ViewData("Name", "iris")
		// render the template with the file name relative to the './templates'.
		// file extension is OPTIONAL.
		if err := ctx.View("hi.html"); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	})

	app.Get("/example_map", func(ctx iris.Context) {
		examplePage := iris.Map{
			"Name":   "Example Name",
			"Age":    42,
			"Items":  []string{"Example slice entry 1", "entry 2", "entry 3"},
			"Map":    iris.Map{"map key": "map value", "other key": "other value"},
			"Nested": iris.Map{"Title": "Iris E-Book", "Pages": 620},
		}

		if err := ctx.View("example.html", examplePage); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	})

	app.Get("/example_struct", func(ctx iris.Context) {
		type book struct {
			Title string
			Pages int
		}

		var examplePage = struct {
			Name   string
			Age    int
			Items  []string
			Map    map[string]interface{}
			Nested book
		}{
			"Example Name",
			42,
			[]string{"Example slice entry 1", "entry 2", "entry 3"},
			iris.Map{"map key": "map value", "other key": "other value"},
			book{
				"Iris E-Book",
				620,
			},
		}

		if err := ctx.View("example.html", examplePage); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	})

	app.Get("/functions", func(ctx iris.Context) {
		var functionsPage = struct {
			// A function.
			Now func() time.Time
			// A struct field which contains methods.
			Ctx iris.Context
		}{
			Now: time.Now,
			Ctx: ctx,
		}

		if err := ctx.View("functions.html", functionsPage); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	})

	// http://localhost:8080/
	app.Listen(":8080")
}
