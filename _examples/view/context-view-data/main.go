package main

import (
	"time"

	"github.com/kataras/iris"
)

const (
	DefaultTitle  = "My Awesome Site"
	DefaultLayout = "layouts/layout.html"
)

func main() {
	app := iris.New()
	// output startup banner and error logs on os.Stdout

	// set the view engine target to ./templates folder
	app.RegisterView(iris.HTML("./templates", ".html").Reload(true))

	app.Use(func(ctx iris.Context) {
		// set the title, current time and a layout in order to be used if and when the next handler(s) calls the .Render function
		ctx.ViewData("Title", DefaultTitle)
		now := time.Now().Format(ctx.Application().ConfigurationReadOnly().GetTimeFormat())
		ctx.ViewData("CurrentTime", now)
		ctx.ViewLayout(DefaultLayout)

		ctx.Next()
	})

	app.Get("/", func(ctx iris.Context) {
		ctx.ViewData("BodyMessage", "a sample text here... setted by the route handler")
		if err := ctx.View("index.html"); err != nil {
			ctx.Application().Logger().Infof(err.Error())
		}
	})

	app.Get("/about", func(ctx iris.Context) {
		ctx.ViewData("Title", "My About Page")
		ctx.ViewData("BodyMessage", "about text here... setted by the route handler")

		// same file, just to keep things simple.
		if err := ctx.View("index.html"); err != nil {
			ctx.Application().Logger().Infof(err.Error())
		}
	})

	// http://localhost:8080
	// http://localhost:8080/about
	app.Run(iris.Addr(":8080"))
}

// Notes: ViewData("", myCustomStruct{}) will set this myCustomStruct value as a root binding data,
// so any View("other", "otherValue") will probably fail.
// To clear the binding data: ctx.Set(ctx.Application().ConfigurationReadOnly().GetViewDataContextKey(), nil)
