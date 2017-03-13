package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func main() {
	app := iris.New()
	app.Adapt(httprouter.New())

	userAges := map[string]int{
		"Alice":  25,
		"Bob":    30,
		"Claire": 29,
	}

	// Equivalent with app.HandleFunc("GET", ...)
	app.Get("/users/:name", func(ctx *iris.Context) {
		name := ctx.Param("name")
		age := userAges[name]

		ctx.Writef("%s is %d years old!", name, age)
	})

	app.Listen(":8080")
}
