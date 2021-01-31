package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<h1>Hello World!</h1>")
	})

	// http://localhost:8080
	// Identical to: app.Run(iris.Addr(":8080"))

	app.Listen(":8080")
	// To listen using keep alive tcp connection listener,
	// set the KeepAlive duration configuration instead:
	// app.Listen(":8080", iris.WithKeepAlive(3*time.Minute))
}
