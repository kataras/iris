package main

import "github.com/kataras/iris"

func main() {
	app := iris.New()
	// or app.Use(before) and app.Done(after).
	app.Get("/", before, mainHandler, after)

	// Use registers a middleware(prepend handlers) to all party's, and its children that will be registered
	// after.
	//
	// (`app` is the root children so those use and done handlers will be registered everywhere)
	app.Use(func(ctx iris.Context) {
		println(`before the party's routes and its children,
but this is not applied to the '/' route
because it's registered before the middleware, order matters.`)
		ctx.Next()
	})

	app.Done(func(ctx iris.Context) {
		println("this is executed always last, if the previous handler calls the `ctx.Next()`, it's the reverse of `.Use`")
		message := ctx.Values().GetString("message")
		println("message: " + message)
	})

	app.Get("/home", func(ctx iris.Context) {
		ctx.HTML("<h1> Home </h1>")
		ctx.Values().Set("message", "this is the home message, ip: "+ctx.RemoteAddr())
		ctx.Next() // call the done handlers.
	})

	child := app.Party("/child")
	child.Get("/", func(ctx iris.Context) {
		ctx.Writef(`this is the localhost:8080/child route.
All Use and Done handlers that are registered to the parent party,
are applied here as well.`)
		ctx.Next() // call the done handlers.
	})

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
