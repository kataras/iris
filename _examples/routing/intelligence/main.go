package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()
	app.Get("/home", handler)
	app.Get("/contact", handler)
	app.Get("/contract", handler)

	// http://localhost:8080/home
	// http://localhost:8080/hom
	//
	// http://localhost:8080/contact
	// http://localhost:8080/cont
	//
	// http://localhost:8080/contract
	// http://localhost:8080/contr
	app.Listen(":8080", iris.WithPathIntelligence)
}

func handler(ctx iris.Context) {
	ctx.Writef("Path: %s", ctx.Path())
}
