package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()

	// with default template funcs:
	//
	// - {{ urlpath "mynamedroute" "pathParameter_ifneeded" }}
	// - {{ render "header.html" }}
	// - {{ render_r "header.html" }} // partial relative path to current page
	// - {{ yield }}
	// - {{ current }}
	app.RegisterView(iris.HTML("./templates", ".html").
		Reload(true)) // Set Reload false to production.

	app.Get("/", func(ctx iris.Context) {
		// enable compression based on Accept-Encoding (e.g. "gzip"),
		// alternatively: app.Use(iris.Compression).
		ctx.CompressWriter(true)
		// the .Name inside the ./templates/hi.html.
		ctx.ViewData("Name", "iris")
		// render the template with the file name relative to the './templates'.
		// file extension is OPTIONAL.
		ctx.View("hi.html")
	})

	app.Get("/example_map", func(ctx iris.Context) {
		ctx.View("example.html", iris.Map{
			"Name":   "Example Name",
			"Age":    42,
			"Items":  []string{"Example slice entry 1", "entry 2", "entry 3"},
			"Map":    iris.Map{"map key": "map value", "other key": "other value"},
			"Nested": iris.Map{"Title": "Iris E-Book", "Pages": 620},
		})
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

		ctx.View("example.html", examplePage)
	})

	// http://localhost:8080/
	app.Listen(":8080")
}
