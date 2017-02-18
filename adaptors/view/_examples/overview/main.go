package main

import (
	"encoding/xml"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/gorillamux"
	"gopkg.in/kataras/iris.v6/adaptors/view"
)

// ExampleXML just a test struct to view represents xml content-type
type ExampleXML struct {
	XMLName xml.Name `xml:"example"`
	One     string   `xml:"one,attr"`
	Two     string   `xml:"two,attr"`
}

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger())
	app.Adapt(gorillamux.New())

	app.Get("/data", func(ctx *iris.Context) {
		ctx.Data(iris.StatusOK, []byte("Some binary data here."))
	})

	app.Get("/text", func(ctx *iris.Context) {
		ctx.Text(iris.StatusOK, "Plain text here")
	})

	app.Get("/json", func(ctx *iris.Context) {
		ctx.JSON(iris.StatusOK, map[string]string{"hello": "json"}) // or myjsonStruct{hello:"json}
	})

	app.Get("/jsonp", func(ctx *iris.Context) {
		ctx.JSONP(iris.StatusOK, "callbackName", map[string]string{"hello": "jsonp"})
	})

	app.Get("/xml", func(ctx *iris.Context) {
		ctx.XML(iris.StatusOK, ExampleXML{One: "hello", Two: "xml"}) // or iris.Map{"One":"hello"...}
	})

	app.Get("/markdown", func(ctx *iris.Context) {
		ctx.Markdown(iris.StatusOK, "# Hello Dynamic Markdown Iris")
	})

	app.Adapt(view.HTML("./templates", ".html"))
	app.Get("/template", func(ctx *iris.Context) {

		ctx.MustRender(
			"hi.html",                // the file name of the template relative to the './templates'
			iris.Map{"Name": "Iris"}, // the .Name inside the ./templates/hi.html
			iris.Map{"gzip": false},  // enable gzip for big files
		)

	})

	// ------ first customization without even the need of *Context or a Handler--------
	//
	// Custom new content-/type:
	//	app.Adapt(iris.RenderPolicy(func(out io.Writer, name string, binding interface{}, options ...map[string]interface{}) (error, bool) {
	//		if name == "customcontent-type" {
	//
	//			// some very advanced things here:
	//			out.Write([]byte(binding.(string)))
	//			return nil, true
	//		}
	//		return nil, false
	//	}))
	//
	//	app.Get("/custom", func(ctx *iris.Context) {
	//		ctx.RenderWithStatus(iris.StatusOK, // or MustRender
	//			"customcontent-type",
	//			"my custom content here!",
	//		)
	//	})
	//
	// ---- second -----------------------------------------------------------------------
	//
	// Override the defaults (the json,xml,jsonp,text,data and so on), an existing  content-type:
	//	app.Adapt(iris.RenderPolicy(func(out io.Writer, name string, binding interface{}, options ...map[string]interface{}) (error, bool) {
	//		if name == "text/plain" {
	//			out.Write([]byte("From the custom text/plain renderer: " + binding.(string)))
	//			return nil, true
	//		}
	//
	//		return nil, false
	//	}))
	// // the context.Text's behaviors was changed now by your custom renderer.
	//

	app.Listen(":8080")
}
