package main

import "github.com/kataras/iris"

func main() {
	app := iris.New()
	app.Get("/", before, mainHandler, after)
	app.Run(iris.Addr(":8080"))
}

func before(ctx iris.Context) {
	shareInformation := "this is a sharable information between handlers"

	requestPath := ctx.Path()
	println("Before the mainHandler: " + requestPath)

	ctx.Values().Set("info", shareInformation)
	ctx.Next() // execute the next handler, in this case the main one.
}

func after(ctx iris.Context) {
	println("After the mainHandler")
}

func mainHandler(ctx iris.Context) {
	println("Inside mainHandler")

	// take the info from the "before" handler.
	info := ctx.Values().GetString("info")

	// write something to the client as a response.
	ctx.HTML("<h1>Response</h1>")
	ctx.HTML("<br/> Info: " + info)

	ctx.Next() // execute the "after".
}
