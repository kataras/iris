package main

import (
	"github.com/kataras/iris/_examples/tutorial/mvc/controllers"
	"github.com/kataras/iris/_examples/tutorial/mvc/persistence"

	"github.com/kataras/iris"
)

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./views", ".html"))

	db := persistence.OpenDatabase("a fake db")

	controllers.RegisterController(app, "/", new(controllers.Index))

	controllers.RegisterController(app, "/user/{userid:int}",
		controllers.NewUserController(db))

	// http://localhost/
	// http://localhost:8080/user/42
	app.Run(iris.Addr(":8080"))
}
