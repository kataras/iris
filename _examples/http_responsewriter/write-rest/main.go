package main

import (
	"encoding/xml"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

// User bind struct
type User struct {
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	City      string `json:"city"`
	Age       int    `json:"age"`
}

// ExampleXML just a test struct to view represents xml content-type
type ExampleXML struct {
	XMLName xml.Name `xml:"example"`
	One     string   `xml:"one,attr"`
	Two     string   `xml:"two,attr"`
}

func main() {
	app := iris.New()

	// Read
	app.Post("/decode", func(ctx iris.Context) {
		var user User
		ctx.ReadJSON(&user)

		ctx.Writef("%s %s is %d years old and comes from %s!", user.Firstname, user.Lastname, user.Age, user.City)
	})

	// Write
	app.Get("/encode", func(ctx iris.Context) {
		peter := User{
			Firstname: "John",
			Lastname:  "Doe",
			City:      "Neither FBI knows!!!",
			Age:       25,
		}

		// Manually setting a content type: ctx.ContentType("application/javascript")
		ctx.JSON(peter)
	})

	// Other content types,

	app.Get("/binary", func(ctx iris.Context) {
		// useful when you want force-download of contents of raw bytes form.
		ctx.Binary([]byte("Some binary data here."))
	})

	app.Get("/text", func(ctx iris.Context) {
		ctx.Text("Plain text here")
	})

	app.Get("/json", func(ctx iris.Context) {
		ctx.JSON(map[string]string{"hello": "json"}) // or myjsonStruct{hello:"json}
	})

	app.Get("/jsonp", func(ctx iris.Context) {
		ctx.JSONP(map[string]string{"hello": "jsonp"}, context.JSONP{Callback: "callbackName"})
	})

	app.Get("/xml", func(ctx iris.Context) {
		ctx.XML(ExampleXML{One: "hello", Two: "xml"}) // or iris.Map{"One":"hello"...}
	})

	app.Get("/markdown", func(ctx iris.Context) {
		ctx.Markdown([]byte("# Hello Dynamic Markdown -- iris"))
	})

	// http://localhost:8080/decode
	// http://localhost:8080/encode
	//
	// http://localhost:8080/binary
	// http://localhost:8080/text
	// http://localhost:8080/json
	// http://localhost:8080/jsonp
	// http://localhost:8080/xml
	// http://localhost:8080/markdown
	//
	// `iris.WithOptimizations` is an optional configurator,
	// if passed to the `Run` then it will ensure that the application
	// response to the client as fast as possible.
	//
	//
	// `iris.WithoutServerError` is an optional configurator,
	// if passed to the `Run` then it will not print its passed error as an actual server error.
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed), iris.WithOptimizations)
}
