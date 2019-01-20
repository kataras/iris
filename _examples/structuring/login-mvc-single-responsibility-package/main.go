package main

import (
	"time"

	"github.com/kataras/iris/_examples/structuring/login-mvc-single-responsibility-package/user"

	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"github.com/kataras/iris/sessions"
)

func main() {
	app := iris.New()
	// You got full debug messages, useful when using MVC and you want to make
	// sure that your code is aligned with the Iris' MVC Architecture.
	app.Logger().SetLevel("debug")

	app.RegisterView(iris.HTML("./views", ".html").Layout("shared/layout.html"))

	app.StaticWeb("/public", "./public")

	mvc.Configure(app, configureMVC)

	// http://localhost:8080/user/register
	// http://localhost:8080/user/login
	// http://localhost:8080/user/me
	// http://localhost:8080/user/logout
	// http://localhost:8080/user/1
	app.Run(iris.Addr(":8080"), configure)
}

func configureMVC(app *mvc.Application) {
	manager := sessions.New(sessions.Config{
		Cookie:  "sessioncookiename",
		Expires: 24 * time.Hour,
	})

	userApp := app.Party("/user")
	userApp.Register(
		user.NewDataSource(),
		manager.Start,
	)
	userApp.Handle(new(user.Controller))
}

func configure(app *iris.Application) {
	app.Configure(
		iris.WithoutServerError(iris.ErrServerClosed),
	)
}
