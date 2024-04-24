package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("Hello %s", "world")
	})

	// This is an Iris-only feature across all web frameworks
	// in every programming language for years.
	// Dynamic Route Path Parameters Functions.
	// Set min length characters to 2.
	// Prefix of the username is '@'
	// Otherwise 404.
	//
	// You can also use the regexp(...) function for more advanced expressions.
	app.Get("/{username:string min(2) prefix(@)}", func(ctx iris.Context) {
		username := ctx.Params().Get("username")[1:]
		ctx.Writef("Username is %s", username)
	})

	// http://localhost:8080          -> FOUND (Hello world)
	// http://localhost:8080/other    -> NOT FOUND
	// http://localhost:8080/@        -> NOT FOUND
	// http://localhost:8080/@kataras -> FOUND (username is kataras)
	app.Listen(":8080")
}
