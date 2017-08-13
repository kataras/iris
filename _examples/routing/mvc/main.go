// +build !go1.9

package main

import (
	"github.com/kataras/iris/_examples/routing/mvc/controllers"
	"github.com/kataras/iris/_examples/routing/mvc/persistence"

	"github.com/kataras/iris"
)

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./views", ".html"))

	db := persistence.OpenDatabase("a fake db")

	app.Controller("/", new(controllers.Index))

	app.Controller("/user/{userid:int}", controllers.NewUserController(db))

	// http://localhost:8080/
	// http://localhost:8080/user/42
	app.Run(iris.Addr(":8080"))
}
