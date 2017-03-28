// this example will show you how you can set per-request data for a template outside of the main handler which calls
// the .Render, via middleware.
//
// Remember: .Render has the "binding" argument which can be used to send data to the template at any case.
package main

import (
	"time"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/view"
)

const (
	DefaultTitle  = "My Awesome Site"
	DefaultLayout = "layouts/layout.html"
)

func main() {
	app := iris.New()
	// output startup banner and error logs on os.Stdout
	app.Adapt(iris.DevLogger())
	// set the router, you can choose gorillamux too
	app.Adapt(httprouter.New())
	// set the view engine target to ./templates folder
	app.Adapt(view.HTML("./templates", ".html").Reload(true))

	app.UseFunc(func(ctx *iris.Context) {
		// set the title, current time and a layout in order to be used if and when the next handler(s) calls the .Render function
		ctx.ViewData("Title", DefaultTitle)
		now := time.Now().Format(app.Config.TimeFormat)
		ctx.ViewData("CurrentTime", now)
		ctx.ViewLayout(DefaultLayout)

		ctx.Next()
	})

	app.Get("/", func(ctx *iris.Context) {
		ctx.ViewData("BodyMessage", "a sample text here... setted by the route handler")
		if err := ctx.Render("index.html", nil); err != nil {
			app.Log(iris.DevMode, err.Error())
		}
	})

	app.Get("/about", func(ctx *iris.Context) {
		ctx.ViewData("Title", "My About Page")
		ctx.ViewData("BodyMessage", "about text here... setted by the route handler")

		// same file, just to keep things simple.
		if err := ctx.Render("index.html", nil); err != nil {
			app.Log(iris.DevMode, err.Error())
		}
	})

	// Open localhost:8080 and localhost:8080/about
	app.Listen(":8080")
}
