// Package main contains three different ways to render content based on the client's accepted.
package main

import "github.com/kataras/iris/v12"

type testdata struct {
	Name string `json:"name" xml:"Name"`
	Age  int    `json:"age" xml:"Age"`
}

func newApp() *iris.Application {
	app := iris.New()
	app.Logger().SetLevel("debug")

	// app.Use(func(ctx iris.Context) {
	// 	requestedMime := ctx.URLParamDefault("type", "application/json")
	//
	//  ctx.Negotiation().Accept.Override().MIME(requestedMime, nil)
	// 	ctx.Next()
	// })

	app.Get("/resource", func(ctx iris.Context) {
		data := testdata{
			Name: "test name",
			Age:  26,
		}

		// Server allows response only JSON and XML. These values
		// are compared with the clients mime needs. Iris comes with default mime types responses
		// but you can add a custom one by the `Negotiation().Mime(mime, content)` method,
		// same for the "accept".
		// You can also pass a custom ContentSelector(mime string) or ContentNegotiator to the
		// `Context.Negotiate` method if you want to perform more advanced things.
		//
		//
		// By-default the client accept mime is retrieved by the "Accept" header
		// Indeed you can override or update it by `Negotiation().Accept.XXX` i.e
		// ctx.Negotiation().Accept.Override().XML()
		//
		// All these values can change inside middlewares, the `Negotiation().Override()` and `.Accept.Override()`
		// can override any previously set values.
		// Order matters, if the client accepts anything (*/*)
		// then the first prioritized mime's response data will be rendered.
		ctx.Negotiation().JSON().XML()
		// Accept-Charset vs:
		ctx.Negotiation().Charset("utf-8", "iso-8859-7")
		// Alternatively you can define the content/data per mime type
		// anywhere in the handlers chain using the optional "v" variadic
		// input argument of the Context.Negotiation().JSON,XML,YAML,Binary,Text,HTML(...) and e.t.c
		// example (order matters):
		// ctx.Negotiation().JSON(data).XML(data).Any("content for */*")
		// ctx.Negotiate(nil)

		// if not nil passed in the `Context.Negotiate` method
		// then it overrides any contents made by the negotitation builder above.
		_, err := ctx.Negotiate(data)
		if err != nil {
			ctx.Writef("%v", err)
		}
	})

	app.Get("/resource2", func(ctx iris.Context) {
		jsonAndXML := testdata{
			Name: "test name",
			Age:  26,
		}

		// I prefer that one, as it gives me the freedom to modify
		// response data per accepted mime content type on middlewares as well.
		ctx.Negotiation().
			JSON(jsonAndXML).
			XML(jsonAndXML).
			HTML("<h1>Test Name</h1><h2>Age 26</h2>")

		ctx.Negotiate(nil)
	})

	app.Get("/resource3", func(ctx iris.Context) {
		// If that line is missing and the requested
		// mime type of content is */* or application/xml or application/json
		// then 406 Not Acceptable http error code will be rendered instead.
		//
		// We also add the "gzip" algorithm as an option to encode
		// resources on send.
		ctx.Negotiation().JSON().XML().HTML().EncodingGzip()

		jsonAndXML := testdata{
			Name: "test name",
			Age:  26,
		}

		// Prefer that way instead of the '/resource2' above
		// if "iris.N" is a static one and can be declared
		// outside of a handler.
		ctx.Negotiate(iris.N{
			// Text: for text/plain,
			// Markdown: for text/mardown,
			// Binary: for application/octet-stream,
			// YAML: for application/x-yaml,
			// JSONP: for text/javascript
			// Other: for anything else,
			JSON: jsonAndXML,                          // for application/json
			XML:  jsonAndXML,                          // for application/xml or text/xml
			HTML: "<h1>Test Name</h1><h2>Age 26</h2>", // for text/html
		})
	})

	return app
}

func main() {
	app := newApp()
	app.Listen(":8080")
}
