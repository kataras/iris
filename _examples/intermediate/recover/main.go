package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/middleware/recover"
)

func main() {
	app := iris.New()
	// fast way to enable non-fatal messages to be printed to the user
	// (yes in iris even recover's errors are not fatal because it's restarting,
	// ProdMode messages are only for things that Iris cannot continue at all,
	// these are logged by-default but you can change that behavior too by passing a different LoggerPolicy to the .Adapt)
	app.Adapt(iris.DevLogger())
	// adapt a router, you can use gorillamux too
	app.Adapt(httprouter.New())

	// use this recover(y) middleware
	app.Use(recover.New())

	i := 0
	// let's simmilate a panic every next request
	app.Get("/", func(ctx *iris.Context) {
		i++
		if i%2 == 0 {
			panic("a panic here")
		}
		ctx.Writef("Hello, refresh one time more to get panic!")
	})

	// http://localhost:8080
	app.Listen(":8080")
}
