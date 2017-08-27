package main

import (
	"time"

	"github.com/kataras/iris/_examples/mvc/login-example/user"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	app.RegisterView(iris.HTML("./views", ".html").Layout("shared/layout.html"))

	app.StaticWeb("/public", "./public")

	manager := sessions.New(sessions.Config{
		Cookie:  "sessioncookiename",
		Expires: 24 * time.Hour,
	})
	users := user.NewDataSource()

	app.Controller("/user", new(user.Controller), manager, users)

	// http://localhost:8080/user/register
	// http://localhost:8080/user/login
	// http://localhost:8080/user/me
	// http://localhost:8080/user/logout
	// http://localhost:8080/user/1
	app.Run(iris.Addr(":8080"), configure)
}

func configure(app *iris.Application) {
	app.Configure(
		iris.WithoutServerError(iris.ErrServerClosed),
		iris.WithCharset("UTF-8"),
	)
}
