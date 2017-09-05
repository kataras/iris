package main

import "github.com/kataras/iris"

func main() {
	app := iris.New()
	// register the "before" handler as the first handler which will be executed
	// on all domain's routes.
	// or use the `UseGlobal` to register a middleware which will fire across subdomains.
	app.Use(before)
	// register the "after" handler as the last handler which will be executed
	// after all domain's routes' handler(s).
	app.Done(after)

	// register our routes.
	app.Get("/", indexHandler)
	app.Get("/contact", contactHandler)

	app.Run(iris.Addr(":8080"))
}

func before(ctx iris.Context) {
	shareInformation := "this is a sharable information between handlers"

	requestPath := ctx.Path()
	println("Before the indexHandler or contactHandler: " + requestPath)

	ctx.Values().Set("info", shareInformation)
	ctx.Next()
}

func after(ctx iris.Context) {
	println("After the indexHandler or contactHandler")
}

func indexHandler(ctx iris.Context) {
	println("Inside indexHandler")

	// take the info from the "before" handler.
	info := ctx.Values().GetString("info")

	// write something to the client as a response.
	ctx.HTML("<h1>Response</h1>")
	ctx.HTML("<br/> Info: " + info)

	ctx.Next() // execute the "after" handler registered via `Done`.
}

func contactHandler(ctx iris.Context) {
	println("Inside contactHandler")

	// write something to the client as a response.
	ctx.HTML("<h1>Contact</h1>")

	ctx.Next() // execute the "after" handler registered via `Done`.
}
