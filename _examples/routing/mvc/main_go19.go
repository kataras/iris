// +build go1.9

package main

import (
	"github.com/kataras/iris/_examples/routing/mvc/controllers"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
	"github.com/kataras/iris/sessions/sessiondb/boltdb"
)

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./views", ".html"))

	sessionDb, _ := boltdb.New("./sessions/sessions.db", 0666, "users")
	sess := sessions.New(sessions.Config{Cookie: "sessionscookieid"})
	sess.UseDatabase(sessionDb.Async(true))

	app.Controller("/", new(controllers.Index))

	app.Controller("/user/{userid:int}", controllers.NewUserController(sess))

	// http://localhost:8080/
	// http://localhost:8080/user/42
	app.Run(iris.Addr(":8080"))
}
