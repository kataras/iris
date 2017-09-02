package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"

	"github.com/betacraft/yaag/irisyaag"
	"github.com/betacraft/yaag/yaag"
)

/*
	go get github.com/betacraft/yaag/...
*/

type myXML struct {
	Result string `xml:"result"`
}

func main() {
	app := iris.New()

	yaag.Init(&yaag.Config{ // <- IMPORTANT, init the middleware.
		On:       true,
		DocTitle: "Iris",
		DocPath:  "apidoc.html",
		BaseUrls: map[string]string{"Production": "", "Staging": ""},
	})
	app.Use(irisyaag.New()) // <- IMPORTANT, register the middleware.

	app.Get("/json", func(ctx context.Context) {
		ctx.JSON(context.Map{"result": "Hello World!"})
	})

	app.Get("/plain", func(ctx context.Context) {
		ctx.Text("Hello World!")
	})

	app.Get("/xml", func(ctx context.Context) {
		ctx.XML(myXML{Result: "Hello World!"})
	})

	app.Get("/complex", func(ctx context.Context) {
		value := ctx.URLParam("key")
		ctx.JSON(context.Map{"value": value})
	})

	// Run our HTTP Server.
	//
	// Documentation of "yaag" doesn't note the follow, but in Iris we are careful on what
	// we provide to you.
	//
	// Each incoming request results on re-generation and update of the "apidoc.html" file.
	// Recommentation:
	// Write tests that calls those handlers, save the generated "apidoc.html".
	// Turn off the yaag middleware when in production.
	//
	// Example usage:
	// Visit all paths and open the generated "apidoc.html" file to see the API's automated docs.
	app.Run(iris.Addr(":8080"))
}
