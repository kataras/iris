package main

import (
	"encoding/xml"

	"github.com/cdren/iris"
	"github.com/cdren/iris/context"
	"github.com/cdren/iris/view"
)

// ExampleXML just a test struct to view represents xml content-type
type ExampleXML struct {
	XMLName xml.Name `xml:"example"`
	One     string   `xml:"one,attr"`
	Two     string   `xml:"two,attr"`
}

func main() {
	app := iris.New()

	// Just some general restful render types, none of these has to do anything with templates.
	app.Get("/binary", func(ctx context.Context) { // useful when you want force-download of contents of raw bytes form.
		ctx.Binary([]byte("Some binary data here."))
	})

	app.Get("/text", func(ctx context.Context) {
		ctx.Text("Plain text here")
	})

	app.Get("/json", func(ctx context.Context) {
		ctx.JSON(map[string]string{"hello": "json"}) // or myjsonStruct{hello:"json}
	})

	app.Get("/jsonp", func(ctx context.Context) {
		ctx.JSONP(map[string]string{"hello": "jsonp"}, context.JSONP{Callback: "callbackName"})
	})

	app.Get("/xml", func(ctx context.Context) {
		ctx.XML(ExampleXML{One: "hello", Two: "xml"}) // or context.Map{"One":"hello"...}
	})

	app.Get("/markdown", func(ctx context.Context) {
		ctx.Markdown([]byte("# Hello Dynamic Markdown -- Iris"))
	})

	//

	// - standard html  | view.HTML(...)
	// - django         | view.Django(...)
	// - pug(jade)      | view.Pug(...)
	// - handlebars     | view.Handlebars(...)
	// - amber          | view.Amber(...)
	// with default template funcs:
	//
	// - {{ urlpath "mynamedroute" "pathParameter_ifneeded" }}
	// - {{ render "header.html" }}
	// - {{ render_r "header.html" }} // partial relative path to current page
	// - {{ yield }}
	// - {{ current }}
	app.AttachView(view.HTML("./templates", ".html"))
	app.Get("/template", func(ctx context.Context) {

		ctx.ViewData("Name", "Iris") // the .Name inside the ./templates/hi.html
		ctx.Gzip(true)               // enable gzip for big files
		ctx.View("hi.html")          // render the template with the file name relative to the './templates'

	})

	// http://localhost:8080/binary
	// http://localhost:8080/text
	// http://localhost:8080/json
	// http://localhost:8080/jsonp
	// http://localhost:8080/xml
	// http://localhost:8080/markdown
	// http://localhost:8080/template
	app.Run(iris.Addr(":8080"))
}
