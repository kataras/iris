// Package main an example on how to naming your routes & use the custom 'url path' Jet Template Engine.
package main

import (
	"fmt"

	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()
	app.RegisterView(iris.Jet("./views", ".jet").Reload(true))

	mypathRoute := app.Get("/mypath", writePathHandler)
	mypathRoute.Name = "my-page1"

	mypath2Route := app.Get("/mypath2/{paramfirst}/{paramsecond}", writePathHandler)
	mypath2Route.Name = "my-page2"

	mypath3Route := app.Get("/mypath3/{paramfirst}/statichere/{paramsecond}", writePathHandler)
	mypath3Route.Name = "my-page3"

	mypath4Route := app.Get("/mypath4/{paramfirst}/statichere/{paramsecond}/{otherparam}/{something:path}", writePathHandler)
	// same as: app.Get("/mypath4/:paramfirst/statichere/:paramsecond/:otherparam/*something", writePathHandler)
	mypath4Route.Name = "my-page4"

	// same with Handle/Func
	mypath5Route := app.Handle("GET", "/mypath5/{paramfirst:int}/statichere/{paramsecond}/{otherparam}/anything/{something:path}", writePathHandlerPage5)
	mypath5Route.Name = "my-page5"

	mypath6Route := app.Get("/mypath6/{paramfirst}/{paramsecond}/statichere/{paramThirdAfterStatic}", writePathHandler)
	mypath6Route.Name = "my-page6"

	app.Get("/", func(ctx iris.Context) {
		// for /mypath6...
		paramsAsArray := []string{"theParam1", "theParam2", "paramThirdAfterStatic"}
		ctx.ViewData("ParamsAsArray", paramsAsArray)
		if err := ctx.View("page.jet"); err != nil {
			ctx.HTML(fmt.Sprintf("<h3>%s</h3>", err.Error()))
			return
		}
	})

	app.Get("/redirect/{namedRoute}", func(ctx iris.Context) {
		routeName := ctx.Params().Get("namedRoute")
		r := app.GetRoute(routeName)
		if r == nil {
			ctx.StatusCode(iris.StatusNotFound)
			ctx.Writef("Route with name %s not found", routeName)
			return
		}

		println("The path of " + routeName + "is: " + r.Path)
		// if routeName == "my-page1"
		// prints: The path of of my-page1 is: /mypath
		// if it's a path which takes named parameters
		// then use "r.ResolvePath(paramValuesHere)"
		ctx.Redirect(r.Path)
		// http://localhost:8080/redirect/my-page1 will redirect to -> http://localhost:8080/mypath
	})

	// http://localhost:8080
	// http://localhost:8080/redirect/my-page1
	app.Listen(":8080")
}

func writePathHandler(ctx iris.Context) {
	ctx.Writef("Hello from %s.", ctx.Path())
}

func writePathHandlerPage5(ctx iris.Context) {
	ctx.Writef("Hello from %s.\nparamfirst(int)=%d", ctx.Path(), ctx.Params().GetIntDefault("paramfirst", 0))
}
