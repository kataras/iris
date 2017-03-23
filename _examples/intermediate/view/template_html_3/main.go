// Package main an example on how to naming your routes & use the custom 'url' HTML Template Engine, same for other template engines.
package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/gorillamux"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/view"
)

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger())
	app.Adapt(view.HTML("./templates", ".html").Reload(true))

	startWithHTTPRouter(app)
	// or uncomment
	// startWithGorillamux()

	app.Listen("localhost:8080")

}

func writePathHandler(ctx *iris.Context) {
	ctx.Writef("Hello from %s.", ctx.Path())
}

func startWithHTTPRouter(app *iris.Framework) {

	app.Adapt(httprouter.New())

	app.Get("/mypath", writePathHandler).ChangeName("my-page1")
	app.Get("/mypath2/:param1/:param2", writePathHandler).ChangeName("my-page2")
	app.Get("/mypath3/:param1/statichere/:param2", writePathHandler).ChangeName("my-page3")
	app.Get("/mypath4/:param1/statichere/:param2/:otherparam/*something", writePathHandler).ChangeName("my-page4")

	// same with Handle/Func
	app.HandleFunc("GET", "/mypath5/:param1/statichere/:param2/:otherparam/anything/*something", writePathHandler).ChangeName("my-page5")

	app.Get("/mypath6/:param1/:param2/staticParam/:param3AfterStatic", writePathHandler).ChangeName("my-page6")

	app.Get("/", func(ctx *iris.Context) {
		// for /mypath6...
		paramsAsArray := []string{"param1", "theParam1",
			"param2", "theParam2",
			"param3AfterStatic", "theParam3"}

		if err := ctx.Render("page.html", iris.Map{"ParamsAsArray": paramsAsArray}); err != nil {
			panic(err)
		}
	})

	app.Get("/redirect/:namedRoute", func(ctx *iris.Context) {
		routeName := ctx.Param("namedRoute")

		println("The full uri of " + routeName + "is: " + app.URL(routeName))
		// if routeName == "my-page1"
		// prints: The full uri of my-page1 is: http://127.0.0.1:8080/mypath
		ctx.RedirectTo(routeName)
		// http://127.0.0.1:8080/redirect/my-page1 will redirect to -> http://127.0.0.1:8080/mypath
	})

}

// for gorillamux adaptor is the same thing, the path syntax is the only thing changed ofc.
// Note: Here, we could use app.RouteParam("param1") without even care what router is being used,
//       but I have two examples of the same thing in order to be more understable for you.
func startWithGorillamux(app *iris.Framework) {
	app.Adapt(gorillamux.New())

	app.Get("/mypath", writePathHandler).ChangeName("my-page1")
	app.Get("/mypath2/{param1}/{param2}", writePathHandler).ChangeName("my-page2")
	app.Get("/mypath3/{param1}/statichere/{param2}", writePathHandler).ChangeName("my-page3")
	app.Get("/mypath4/{param1}/statichere/{param2}/{otherparam}/{something:.*}", writePathHandler).ChangeName("my-page4")

	// same with Handle/Func
	app.HandleFunc("GET", "/mypath5/{param1}/statichere/{param2}/{otherparam}/anything/{something:.*}", writePathHandler).ChangeName("my-page5")

	app.Get("/mypath6/{param1}/{param2}/staticParam/{param3AfterStatic}", writePathHandler).ChangeName("my-page6")

	app.Get("/", func(ctx *iris.Context) {
		// for /mypath6...
		paramsAsArray := []string{"param1", "theParam1",
			"param2", "theParam2",
			"param3AfterStatic", "theParam3"}

		if err := ctx.Render("page.html", iris.Map{"ParamsAsArray": paramsAsArray}); err != nil {
			panic(err)
		}
	})

	app.Get("/redirect/{namedRoute}", func(ctx *iris.Context) {
		routeName := ctx.Param("namedRoute")

		println("The full uri of " + routeName + "is: " + app.URL(routeName))
		// if routeName == "my-page1"
		// prints: The full uri of my-page1 is: http://127.0.0.1:8080/mypath
		ctx.RedirectTo(routeName)
		// http://127.0.0.1:8080/redirect/my-page1 will redirect to -> http://127.0.0.1:8080/mypath
	})

	app.Listen("localhost:8080")
}
